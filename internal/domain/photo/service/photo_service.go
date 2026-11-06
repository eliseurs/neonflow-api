package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"

	"neonflow-api/internal/domain/photo"
	"neonflow-api/internal/infra/storage"
)

type PhotoService struct {
	repo  photo.Repository
	store photo.Store
}

func NewPhotoService(repo photo.Repository, store photo.Store) *PhotoService {
	return &PhotoService{repo: repo, store: store}
}

// Upload confere o dono, a etapa, o tamanho e o formato (pelo conteúdo), grava no bucket e registra.
func (s *PhotoService) Upload(ctx context.Context, ownerType, ownerID, stage, userID string, body io.Reader, size int64) (photo.Photo, error) {
	if size <= 0 {
		return photo.Photo{}, photo.ErrEmpty
	}
	if size > photo.MaxSize {
		return photo.Photo{}, photo.ErrTooLarge
	}
	if err := s.checkOwner(ctx, ownerType, ownerID, stage); err != nil {
		return photo.Photo{}, err
	}
	head := make([]byte, 512)
	n, err := io.ReadFull(body, head)
	if err != nil && err != io.ErrUnexpectedEOF {
		return photo.Photo{}, photo.ErrEmpty
	}
	contentType, ext, err := photo.DetectType(head[:n])
	if err != nil {
		return photo.Photo{}, err
	}
	name, err := randomName()
	if err != nil {
		return photo.Photo{}, err
	}
	folder := map[string]string{photo.OwnerVehicle: "veiculos", photo.OwnerBooking: "reservas"}[ownerType]
	key := storage.CleanKey(folder, ownerID, stage, name+"."+ext)
	if err := s.store.Put(ctx, key, io.MultiReader(bytes.NewReader(head[:n]), body), size, contentType); err != nil {
		return photo.Photo{}, err
	}
	p, err := s.repo.Create(ctx, photo.NewPhoto{OwnerType: ownerType, OwnerID: ownerID, Stage: stage, Key: key,
		ContentType: contentType, SizeBytes: int(size), CreatedBy: userID})
	if err != nil {
		_ = s.store.Delete(ctx, key) // sem registro, o arquivo não fica órfão no bucket
		return photo.Photo{}, err
	}
	return s.withURL(ctx, p)
}

func (s *PhotoService) checkOwner(ctx context.Context, ownerType, ownerID, stage string) error {
	switch ownerType {
	case photo.OwnerVehicle:
		ok, err := s.repo.VehicleExists(ctx, ownerID)
		if err != nil {
			return err
		}
		if !ok {
			return photo.ErrOwnerNotFound
		}
		return nil
	case photo.OwnerBooking:
		if stage != photo.StageDelivery && stage != photo.StageReturn {
			return photo.ErrInvalidStage
		}
		bookingStage, err := s.repo.BookingStage(ctx, ownerID)
		if err != nil {
			return err
		}
		if !photo.StageAllowed(stage, bookingStage) {
			return photo.ErrWrongStage
		}
		return nil
	}
	return photo.ErrOwnerNotFound
}

// List devolve as fotos com links temporários de leitura.
func (s *PhotoService) List(ctx context.Context, ownerType, ownerID, stage string) ([]photo.Photo, error) {
	if ownerType == photo.OwnerBooking && stage != "" && stage != photo.StageDelivery && stage != photo.StageReturn {
		return nil, photo.ErrInvalidStage
	}
	list, err := s.repo.List(ctx, ownerType, ownerID, stage)
	if err != nil {
		return nil, err
	}
	for i := range list {
		if list[i], err = s.withURL(ctx, list[i]); err != nil {
			return nil, err
		}
	}
	return list, nil
}

func (s *PhotoService) Delete(ctx context.Context, id string) error {
	p, err := s.repo.Delete(ctx, id)
	if err != nil {
		return err
	}
	if p.OwnerType == photo.OwnerVehicle {
		for _, w := range photo.VariantWidths {
			_ = s.store.Delete(ctx, photo.VariantKey(p.Key, w)) // versões que nunca foram geradas não existem
		}
	}
	return s.store.Delete(ctx, p.Key)
}

// PublicFleet: os carros do site, com o id da foto de capa.
func (s *PhotoService) PublicFleet(ctx context.Context) ([]photo.Car, error) {
	return s.repo.PublicFleet(ctx)
}

// Variant entrega a versão da foto do veículo na largura pedida. Na primeira vez gera a partir da
// original e guarda no bucket; depois só lê.
func (s *PhotoService) Variant(ctx context.Context, id string, width int) ([]byte, error) {
	if !photo.VariantAllowed(width) {
		return nil, photo.ErrInvalidWidth
	}
	p, err := s.repo.VehiclePhoto(ctx, id)
	if err != nil {
		return nil, err
	}
	key := photo.VariantKey(p.Key, width)
	data, err := s.store.Get(ctx, key)
	if err == nil {
		return data, nil
	}
	if !errors.Is(err, storage.ErrNotFound) {
		return nil, err
	}
	original, err := s.store.Get(ctx, p.Key)
	if err != nil {
		return nil, err
	}
	out, err := photo.Resize(original, width)
	if err != nil {
		return nil, err
	}
	// Guardar é otimização: se falhar, a próxima visita gera de novo.
	_ = s.store.Put(ctx, key, bytes.NewReader(out), int64(len(out)), "image/jpeg")
	return out, nil
}

func (s *PhotoService) SetCover(ctx context.Context, id string) (photo.Photo, error) {
	p, err := s.repo.SetCover(ctx, id)
	if err != nil {
		return p, err
	}
	return s.withURL(ctx, p)
}

func (s *PhotoService) withURL(ctx context.Context, p photo.Photo) (photo.Photo, error) {
	u, err := s.store.URL(ctx, p.Key, photo.LinkTTL)
	if err != nil {
		return p, err
	}
	p.URL = u
	return p, nil
}

func randomName() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("random name: %w", err)
	}
	return hex.EncodeToString(b), nil
}
