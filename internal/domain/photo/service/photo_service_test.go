package service

import (
	"bytes"
	"context"
	"errors"
	"image"
	stdjpeg "image/jpeg"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"neonflow-api/internal/domain/photo"
	"neonflow-api/internal/infra/storage"
)

var jpeg = []byte{0xFF, 0xD8, 0xFF, 0xE0, 0, 0x10, 'J', 'F', 'I', 'F', 0}

type fakeRepo struct {
	stage      string
	stageErr   error
	vehicle    bool
	vehicleErr error
	created    photo.NewPhoto
	createErr  error
	list       []photo.Photo
	listErr    error
	deleted    photo.Photo
	deleteErr  error
	coverErr   error
	fleet      []photo.Car
	vphoto     photo.Photo
	vphotoErr  error
}

func (f *fakeRepo) BookingStage(context.Context, string) (string, error) { return f.stage, f.stageErr }
func (f *fakeRepo) VehicleExists(context.Context, string) (bool, error) {
	return f.vehicle, f.vehicleErr
}
func (f *fakeRepo) Create(_ context.Context, n photo.NewPhoto) (photo.Photo, error) {
	f.created = n
	return photo.Photo{ID: "p1", Key: n.Key, OwnerType: n.OwnerType}, f.createErr
}
func (f *fakeRepo) List(context.Context, string, string, string) ([]photo.Photo, error) {
	return f.list, f.listErr
}
func (f *fakeRepo) Delete(context.Context, string) (photo.Photo, error) {
	return f.deleted, f.deleteErr
}
func (f *fakeRepo) PublicFleet(context.Context) ([]photo.Car, error) { return f.fleet, nil }
func (f *fakeRepo) PublicCar(_ context.Context, id string) (photo.Car, error) {
	return photo.Car{ID: id, PhotoIDs: []string{"p1", "p2"}}, nil
}
func (f *fakeRepo) VehiclePhoto(context.Context, string) (photo.Photo, error) {
	return f.vphoto, f.vphotoErr
}
func (f *fakeRepo) SetCover(_ context.Context, id string) (photo.Photo, error) {
	return photo.Photo{ID: id, Key: "k"}, f.coverErr
}

type fakeStore struct {
	put     map[string][]byte
	deleted []string
	putErr  error
	urlErr  error
	getErr  error
	gets    []string
}

func (s *fakeStore) Put(_ context.Context, key string, body io.Reader, _ int64, _ string) error {
	if s.putErr != nil {
		return s.putErr
	}
	b, _ := io.ReadAll(body)
	s.put[key] = b
	return nil
}
func (s *fakeStore) Get(_ context.Context, key string) ([]byte, error) {
	s.gets = append(s.gets, key)
	if s.getErr != nil {
		return nil, s.getErr
	}
	b, ok := s.put[key]
	if !ok {
		return nil, storage.ErrNotFound
	}
	return b, nil
}
func (s *fakeStore) URL(_ context.Context, key string, ttl time.Duration) (string, error) {
	return "https://bucket.test/" + key + "?ttl=" + ttl.String(), s.urlErr
}
func (s *fakeStore) Delete(_ context.Context, key string) error {
	s.deleted = append(s.deleted, key)
	return nil
}

func TestUpload(t *testing.T) {
	ctx := context.Background()
	repo := &fakeRepo{vehicle: true, stage: "devolvida"}
	store := &fakeStore{put: map[string][]byte{}}
	svc := NewPhotoService(repo, store)
	body := append(append([]byte{}, jpeg...), bytes.Repeat([]byte("x"), 1000)...)

	p, err := svc.Upload(ctx, photo.OwnerVehicle, "v1", "", "u1", bytes.NewReader(body), int64(len(body)))
	require.NoError(t, err)
	assert.Regexp(t, `^veiculos/v1/[0-9a-f]{32}\.jpg$`, repo.created.Key)
	assert.Equal(t, "image/jpeg", repo.created.ContentType)
	assert.Equal(t, body, store.put[repo.created.Key], "o arquivo chega inteiro, com o cabeçalho lido")
	assert.Contains(t, p.URL, "ttl=15m0s")

	_, err = svc.Upload(ctx, photo.OwnerBooking, "b1", photo.StageReturn, "u1", bytes.NewReader(body), int64(len(body)))
	require.NoError(t, err)
	assert.Regexp(t, `^reservas/b1/devolucao/[0-9a-f]{32}\.jpg$`, repo.created.Key)

	cases := []struct {
		name  string
		owner string
		stage string
		body  []byte
		size  int64
		prep  func()
		want  error
	}{
		{"vazio", photo.OwnerVehicle, "", nil, 0, nil, photo.ErrEmpty},
		{"grande", photo.OwnerVehicle, "", body, photo.MaxSize + 1, nil, photo.ErrTooLarge},
		{"formato", photo.OwnerVehicle, "", []byte("%PDF-1.7 texto"), 14, nil, photo.ErrInvalidType},
		{"veículo inexistente", photo.OwnerVehicle, "", body, int64(len(body)), func() { repo.vehicle = false }, photo.ErrOwnerNotFound},
		{"etapa inválida", photo.OwnerBooking, "outra", body, int64(len(body)), nil, photo.ErrInvalidStage},
		{"etapa errada", photo.OwnerBooking, photo.StageDelivery, body, int64(len(body)), nil, photo.ErrWrongStage},
		{"reserva inexistente", photo.OwnerBooking, photo.StageReturn, body, int64(len(body)), func() { repo.stageErr = photo.ErrOwnerNotFound }, photo.ErrOwnerNotFound},
		{"dono desconhecido", "cliente", "", body, int64(len(body)), nil, photo.ErrOwnerNotFound},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo.vehicle, repo.stageErr = true, nil
			if tc.prep != nil {
				tc.prep()
			}
			_, err := svc.Upload(ctx, tc.owner, "x", tc.stage, "u1", bytes.NewReader(tc.body), tc.size)
			assert.ErrorIs(t, err, tc.want)
		})
	}

	// Falha ao registrar: o arquivo enviado é apagado do bucket.
	repo.vehicle, repo.createErr = true, errors.New("db")
	_, err = svc.Upload(ctx, photo.OwnerVehicle, "v1", "", "u1", bytes.NewReader(body), int64(len(body)))
	assert.Error(t, err)
	assert.Equal(t, repo.created.Key, store.deleted[len(store.deleted)-1])
	repo.createErr = nil

	store.putErr = errors.New("bucket")
	_, err = svc.Upload(ctx, photo.OwnerVehicle, "v1", "", "u1", bytes.NewReader(body), int64(len(body)))
	assert.Error(t, err)
	store.putErr = nil
	repo.vehicleErr = errors.New("db")
	_, err = svc.Upload(ctx, photo.OwnerVehicle, "v1", "", "u1", bytes.NewReader(body), int64(len(body)))
	assert.Error(t, err)
	_, err = svc.Upload(ctx, photo.OwnerVehicle, "v1", "", "u1", strings.NewReader(""), 10)
	assert.Error(t, err)
}

func TestListDeleteCover(t *testing.T) {
	ctx := context.Background()
	repo := &fakeRepo{list: []photo.Photo{{ID: "p1", Key: "veiculos/v1/a.jpg"}}, deleted: photo.Photo{Key: "veiculos/v1/a.jpg"}}
	store := &fakeStore{put: map[string][]byte{}}
	svc := NewPhotoService(repo, store)

	list, err := svc.List(ctx, photo.OwnerVehicle, "v1", "")
	require.NoError(t, err)
	assert.Equal(t, "https://bucket.test/veiculos/v1/a.jpg?ttl=15m0s", list[0].URL)
	_, err = svc.List(ctx, photo.OwnerBooking, "b1", "outra")
	assert.ErrorIs(t, err, photo.ErrInvalidStage)
	store.urlErr = errors.New("presign")
	_, err = svc.List(ctx, photo.OwnerVehicle, "v1", "")
	assert.Error(t, err)
	store.urlErr = nil
	repo.listErr = errors.New("db")
	_, err = svc.List(ctx, photo.OwnerVehicle, "v1", "")
	assert.Error(t, err)

	require.NoError(t, svc.Delete(ctx, "p1"))
	assert.Equal(t, []string{"veiculos/v1/a.jpg"}, store.deleted, "foto de reserva não tem versões")
	repo.deleted.OwnerType = photo.OwnerVehicle
	store.deleted = nil
	require.NoError(t, svc.Delete(ctx, "p1"))
	assert.Equal(t, []string{"veiculos/v1/a-w640.jpg", "veiculos/v1/a-w1280.jpg", "veiculos/v1/a-w1920.jpg", "veiculos/v1/a.jpg"}, store.deleted)
	repo.deleted.OwnerType = ""
	repo.deleteErr = photo.ErrNotFound
	assert.ErrorIs(t, svc.Delete(ctx, "x"), photo.ErrNotFound)

	c, err := svc.SetCover(ctx, "p1")
	require.NoError(t, err)
	assert.NotEmpty(t, c.URL)
	repo.coverErr = photo.ErrNotFound
	_, err = svc.SetCover(ctx, "x")
	assert.ErrorIs(t, err, photo.ErrNotFound)
}

func TestVariant(t *testing.T) {
	ctx := context.Background()
	img := image.NewRGBA(image.Rect(0, 0, 2000, 1000))
	var orig bytes.Buffer
	require.NoError(t, jpegEncode(&orig, img))
	repo := &fakeRepo{vphoto: photo.Photo{ID: "p1", Key: "veiculos/v1/a.webp"}, fleet: []photo.Car{{ID: "v1"}}}
	store := &fakeStore{put: map[string][]byte{"veiculos/v1/a.webp": orig.Bytes()}}
	svc := NewPhotoService(repo, store)

	cars, err := svc.PublicFleet(ctx)
	require.NoError(t, err)
	assert.Len(t, cars, 1)
	car, err := svc.PublicCar(ctx, "v1")
	require.NoError(t, err)
	assert.Equal(t, []string{"p1", "p2"}, car.PhotoIDs)

	// Primeira vez: gera da original e guarda.
	out, err := svc.Variant(ctx, "p1", 640)
	require.NoError(t, err)
	cfg, _, err := image.DecodeConfig(bytes.NewReader(out))
	require.NoError(t, err)
	assert.Equal(t, 640, cfg.Width)
	assert.Equal(t, out, store.put["veiculos/v1/a-w640.jpg"])
	// Depois: só lê a versão guardada.
	store.gets = nil
	again, err := svc.Variant(ctx, "p1", 640)
	require.NoError(t, err)
	assert.Equal(t, out, again)
	assert.Equal(t, []string{"veiculos/v1/a-w640.jpg"}, store.gets)

	_, err = svc.Variant(ctx, "p1", 1000)
	assert.ErrorIs(t, err, photo.ErrInvalidWidth)
	repo.vphotoErr = photo.ErrNotFound
	_, err = svc.Variant(ctx, "p9", 640)
	assert.ErrorIs(t, err, photo.ErrNotFound)
	repo.vphotoErr = nil
	store.getErr = errors.New("bucket")
	_, err = svc.Variant(ctx, "p1", 1280)
	assert.Error(t, err)
	store.getErr = nil
	delete(store.put, "veiculos/v1/a.webp")
	_, err = svc.Variant(ctx, "p1", 1920)
	assert.ErrorIs(t, err, storage.ErrNotFound, "sem a original")
	store.put["veiculos/v1/a.webp"] = []byte("corrompida")
	_, err = svc.Variant(ctx, "p1", 1920)
	assert.ErrorIs(t, err, photo.ErrInvalidType)
}

func jpegEncode(w io.Writer, img image.Image) error { return stdjpeg.Encode(w, img, nil) }
