-- Mínimo de diárias por carro, configurável (06/11/2026). Nulo = o mínimo geral de 1 diária (RN-02.12).
ALTER TABLE vehicles ADD COLUMN min_days INT CHECK (min_days BETWEEN 1 AND 365);
