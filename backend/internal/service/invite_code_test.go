package service

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateInviteCode_UsesExpectedCharsetAndContainsLetter(t *testing.T) {
	t.Parallel()

	for i := 0; i < 256; i++ {
		code, err := GenerateInviteCode()
		require.NoError(t, err)
		require.Len(t, code, 8)

		hasLetter := false
		for _, ch := range code {
			require.Truef(t, strings.ContainsRune(inviteCodeAlphabet, ch), "unexpected invite code character: %q", ch)
			if strings.ContainsRune(inviteCodeLetterAlphabet, ch) {
				hasLetter = true
			}
		}
		require.True(t, hasLetter, "invite code should contain at least one letter")
	}
}
