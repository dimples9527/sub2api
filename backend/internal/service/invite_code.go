package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"strings"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	dbuser "github.com/Wei-Shaw/sub2api/ent/user"
)

const inviteCodeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

func NormalizeInviteCode(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

func GenerateInviteCode() (string, error) {
	const codeLen = 8
	buf := make([]byte, codeLen)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate invite code: %w", err)
	}
	out := make([]byte, codeLen)
	for i, b := range buf {
		out[i] = inviteCodeAlphabet[int(b)%len(inviteCodeAlphabet)]
	}
	return string(out), nil
}

func GenerateUniqueInviteCode(ctx context.Context, client *dbent.Client) (string, error) {
	for i := 0; i < 8; i++ {
		code, err := GenerateInviteCode()
		if err != nil {
			return "", err
		}
		if client == nil {
			return code, nil
		}
		exists, err := client.User.Query().Where(dbuser.InviteCodeEQ(code)).Exist(ctx)
		if err != nil {
			return "", fmt.Errorf("check invite code exists: %w", err)
		}
		if !exists {
			return code, nil
		}
	}
	return "", fmt.Errorf("failed to generate unique invite code")
}

func FindInviterByInviteCode(ctx context.Context, client *dbent.Client, code string) (*User, error) {
	if client == nil {
		return nil, ErrInvitationCodeInvalid
	}
	normalized := NormalizeInviteCode(code)
	if normalized == "" {
		return nil, ErrInvitationCodeInvalid
	}
	entity, err := client.User.Query().Where(dbuser.InviteCodeEQ(normalized)).Only(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, ErrInvitationCodeInvalid
		}
		return nil, fmt.Errorf("query inviter by invite code: %w", err)
	}
	return &User{
		ID:         entity.ID,
		Email:      entity.Email,
		Username:   entity.Username,
		InviteCode: entity.InviteCode,
		Status:     entity.Status,
	}, nil
}
