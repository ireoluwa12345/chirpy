package auth

import "github.com/alexedwards/argon2id"

func hashPassword(password string) (string, error) {
	hash, err := argon2id.CreateHash(password, &argon2id.Params{
		Memory:      64 * 1024,
		Iterations:  3,
		Parallelism: 2,
		SaltLength:  16,
		KeyLength:   32,
	})
	return hash, err
}

func verifyPassword(password, hashedPassword string) (bool, error) {
	return argon2id.ComparePasswordAndHash(password, hashedPassword)
}
