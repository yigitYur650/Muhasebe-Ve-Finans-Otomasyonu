package domain

import "errors"

var (
	ErrNotFound                   = errors.New("kayıt bulunamadı")
	ErrPeriodLocked               = errors.New("dönem kilitli olduğu için işlem yapılamaz")
	ErrInvalidAmount              = errors.New("işlem tutarı sıfırdan büyük olmalıdır")
	ErrInvalidDirection           = errors.New("geçersiz işlem yönü (sadece 'in' veya 'out' olabilir)")
	ErrInvalidChannel             = errors.New("geçersiz işlem kanalı")
	ErrInvalidRole                = errors.New("geçersiz üye rolü")
	ErrTransactionNotFound        = errors.New("işlem bulunamadı")
	ErrPeriodNotFound             = errors.New("dönem bulunamadı")
	ErrTenantNotFound             = errors.New("işletme (tenant) bulunamadı")
	ErrUnauthorized               = errors.New("yetkisiz erişim")
	ErrDuplicateIdempotencyKey    = errors.New("bu Idempotency-Key daha önce kullanılmış")
	ErrTransactionAlreadyReversed = errors.New("bu işlem zaten ters kayıt ile iptal edilmiş")
	ErrCannotRemoveLastAdmin      = errors.New("tenant içerisindeki son admin kullanıcısı silinemez")
)
