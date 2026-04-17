package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/mail"
	"strconv"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials      = infraerrors.Unauthorized("INVALID_CREDENTIALS", "invalid email or password")
	ErrUserNotActive           = infraerrors.Forbidden("USER_NOT_ACTIVE", "user is not active")
	ErrEmailExists             = infraerrors.Conflict("EMAIL_EXISTS", "email already exists")
	ErrEmailReserved           = infraerrors.BadRequest("EMAIL_RESERVED", "email is reserved")
	ErrInvalidToken            = infraerrors.Unauthorized("INVALID_TOKEN", "invalid token")
	ErrTokenExpired            = infraerrors.Unauthorized("TOKEN_EXPIRED", "token has expired")
	ErrAccessTokenExpired      = infraerrors.Unauthorized("ACCESS_TOKEN_EXPIRED", "access token has expired")
	ErrTokenTooLarge           = infraerrors.BadRequest("TOKEN_TOO_LARGE", "token too large")
	ErrTokenRevoked            = infraerrors.Unauthorized("TOKEN_REVOKED", "token has been revoked")
	ErrRefreshTokenInvalid     = infraerrors.Unauthorized("REFRESH_TOKEN_INVALID", "invalid refresh token")
	ErrRefreshTokenExpired     = infraerrors.Unauthorized("REFRESH_TOKEN_EXPIRED", "refresh token has expired")
	ErrRefreshTokenReused      = infraerrors.Unauthorized("REFRESH_TOKEN_REUSED", "refresh token has been reused")
	ErrEmailVerifyRequired     = infraerrors.BadRequest("EMAIL_VERIFY_REQUIRED", "email verification is required")
	ErrEmailSuffixNotAllowed   = infraerrors.BadRequest("EMAIL_SUFFIX_NOT_ALLOWED", "email suffix is not allowed")
	ErrRegDisabled             = infraerrors.Forbidden("REGISTRATION_DISABLED", "registration is currently disabled")
	ErrServiceUnavailable      = infraerrors.ServiceUnavailable("SERVICE_UNAVAILABLE", "service temporarily unavailable")
	ErrInvitationCodeRequired  = infraerrors.BadRequest("INVITATION_CODE_REQUIRED", "invitation code is required")
	ErrInvitationCodeInvalid   = infraerrors.BadRequest("INVITATION_CODE_INVALID", "invalid or used invitation code")
	ErrOAuthInvitationRequired = infraerrors.Forbidden("OAUTH_INVITATION_REQUIRED", "invitation code required to complete oauth registration")
)

// maxTokenLength 闄愬埗 token 澶у皬锛岄伩鍏嶈秴闀?header 瑙﹀彂瑙ｆ瀽鏃剁殑寮傚父鍐呭瓨鍒嗛厤銆?
const maxTokenLength = 8192

// refreshTokenPrefix is the prefix for refresh tokens to distinguish them from access tokens.
const refreshTokenPrefix = "rt_"

// JWTClaims JWT杞借嵎鏁版嵁
type JWTClaims struct {
	UserID       int64  `json:"user_id"`
	Email        string `json:"email"`
	Role         string `json:"role"`
	TokenVersion int64  `json:"token_version"` // Used to invalidate tokens on password change
	jwt.RegisteredClaims
}

// AuthService 璁よ瘉鏈嶅姟
type AuthService struct {
	entClient          *dbent.Client
	userRepo           UserRepository
	redeemRepo         RedeemCodeRepository
	refreshTokenCache  RefreshTokenCache
	cfg                *config.Config
	settingService     *SettingService
	emailService       *EmailService
	turnstileService   *TurnstileService
	emailQueueService  *EmailQueueService
	promoService       *PromoService
	defaultSubAssigner DefaultSubscriptionAssigner
}

type DefaultSubscriptionAssigner interface {
	AssignOrExtendSubscription(ctx context.Context, input *AssignSubscriptionInput) (*UserSubscription, bool, error)
}

// NewAuthService 鍒涘缓璁よ瘉鏈嶅姟瀹炰緥
func NewAuthService(
	entClient *dbent.Client,
	userRepo UserRepository,
	redeemRepo RedeemCodeRepository,
	refreshTokenCache RefreshTokenCache,
	cfg *config.Config,
	settingService *SettingService,
	emailService *EmailService,
	turnstileService *TurnstileService,
	emailQueueService *EmailQueueService,
	promoService *PromoService,
	defaultSubAssigner DefaultSubscriptionAssigner,
) *AuthService {
	return &AuthService{
		entClient:          entClient,
		userRepo:           userRepo,
		redeemRepo:         redeemRepo,
		refreshTokenCache:  refreshTokenCache,
		cfg:                cfg,
		settingService:     settingService,
		emailService:       emailService,
		turnstileService:   turnstileService,
		emailQueueService:  emailQueueService,
		promoService:       promoService,
		defaultSubAssigner: defaultSubAssigner,
	}
}

// Register 鐢ㄦ埛娉ㄥ唽锛岃繑鍥瀟oken鍜岀敤鎴?
func (s *AuthService) Register(ctx context.Context, email, password string) (string, *User, error) {
	return s.RegisterWithVerification(ctx, email, password, "", "", "")
}

// RegisterWithVerification 鐢ㄦ埛娉ㄥ唽锛堟敮鎸侀偖浠堕獙璇併€佷紭鎯犵爜鍜岄個璇风爜锛夛紝杩斿洖token鍜岀敤鎴?
func (s *AuthService) RegisterWithVerification(ctx context.Context, email, password, verifyCode, promoCode, invitationCode string) (string, *User, error) {
	if s.settingService == nil || !s.settingService.IsRegistrationEnabled(ctx) {
		return "", nil, ErrRegDisabled
	}

	if isReservedEmail(email) {
		return "", nil, ErrEmailReserved
	}
	if err := s.validateRegistrationEmailPolicy(ctx, email); err != nil {
		return "", nil, err
	}

	var inviter *User
	if s.settingService != nil && s.settingService.IsInvitationCodeEnabled(ctx) {
		if strings.TrimSpace(invitationCode) == "" {
			return "", nil, ErrInvitationCodeRequired
		}
		resolvedInviter, err := s.lookupInviterByCode(ctx, invitationCode)
		if err != nil {
			logger.LegacyPrintf("service.auth", "[Auth] Invalid invitation code during registration: code=%s err=%v", invitationCode, err)
			return "", nil, ErrInvitationCodeInvalid
		}
		inviter = resolvedInviter
	}

	invitationReward := 0.0
	if inviter != nil && s.settingService != nil {
		invitationReward = s.settingService.GetInvitationReward(ctx)
	}

	if s.settingService != nil && s.settingService.IsEmailVerifyEnabled(ctx) {
		if s.emailService == nil {
			logger.LegacyPrintf("service.auth", "%s", "[Auth] Email verification enabled but email service not configured, rejecting registration")
			return "", nil, ErrServiceUnavailable
		}
		if verifyCode == "" {
			return "", nil, ErrEmailVerifyRequired
		}
		if err := s.emailService.VerifyCode(ctx, email, verifyCode); err != nil {
			return "", nil, fmt.Errorf("verify code: %w", err)
		}
	}

	existsEmail, err := s.userRepo.ExistsByEmail(ctx, email)
	if err != nil {
		logger.LegacyPrintf("service.auth", "[Auth] Database error checking email exists: %v", err)
		return "", nil, ErrServiceUnavailable
	}
	if existsEmail {
		return "", nil, ErrEmailExists
	}

	hashedPassword, err := s.HashPassword(password)
	if err != nil {
		return "", nil, fmt.Errorf("hash password: %w", err)
	}

	defaultBalance := s.cfg.Default.UserBalance
	defaultConcurrency := s.cfg.Default.UserConcurrency
	if s.settingService != nil {
		defaultBalance = s.settingService.GetDefaultBalance(ctx)
		defaultConcurrency = s.settingService.GetDefaultConcurrency(ctx)
	}

	user := &User{
		Email:        email,
		PasswordHash: hashedPassword,
		Role:         RoleUser,
		Balance:      defaultBalance,
		Concurrency:  defaultConcurrency,
		Status:       StatusActive,
	}

	if err := s.createUserWithInvitationReward(ctx, user, inviter, invitationReward); err != nil {
		if errors.Is(err, ErrEmailExists) {
			return "", nil, ErrEmailExists
		}
		logger.LegacyPrintf("service.auth", "[Auth] Database error creating user: %v", err)
		return "", nil, ErrServiceUnavailable
	}

	if promoCode != "" && s.promoService != nil && s.settingService != nil && s.settingService.IsPromoCodeEnabled(ctx) {
		if err := s.promoService.ApplyPromoCode(ctx, user.ID, promoCode); err != nil {
			logger.LegacyPrintf("service.auth", "[Auth] Failed to apply promo code for user %d: %v", user.ID, err)
		} else if updatedUser, err := s.userRepo.GetByID(ctx, user.ID); err == nil {
			user = updatedUser
		}
	}

	token, err := s.GenerateToken(user)
	if err != nil {
		return "", nil, fmt.Errorf("generate token: %w", err)
	}

	return token, user, nil
}

// SendVerifyCodeResult 鍙戦€侀獙璇佺爜杩斿洖缁撴灉
type SendVerifyCodeResult struct {
	Countdown int `json:"countdown"` // 鍊掕鏃剁鏁?
}

// SendVerifyCode 鍙戦€侀偖绠遍獙璇佺爜锛堝悓姝ユ柟寮忥級
func (s *AuthService) SendVerifyCode(ctx context.Context, email string) error {
	// 妫€鏌ユ槸鍚﹀紑鏀炬敞鍐岋紙榛樿鍏抽棴锛?
	if s.settingService == nil || !s.settingService.IsRegistrationEnabled(ctx) {
		return ErrRegDisabled
	}

	if isReservedEmail(email) {
		return ErrEmailReserved
	}
	if err := s.validateRegistrationEmailPolicy(ctx, email); err != nil {
		return err
	}

	// 妫€鏌ラ偖绠辨槸鍚﹀凡瀛樺湪
	existsEmail, err := s.userRepo.ExistsByEmail(ctx, email)
	if err != nil {
		logger.LegacyPrintf("service.auth", "[Auth] Database error checking email exists: %v", err)
		return ErrServiceUnavailable
	}
	if existsEmail {
		return ErrEmailExists
	}

	// 鍙戦€侀獙璇佺爜
	if s.emailService == nil {
		return errors.New("email service not configured")
	}

	// 鑾峰彇缃戠珯鍚嶇О
	siteName := "Sub2API"
	if s.settingService != nil {
		siteName = s.settingService.GetSiteName(ctx)
	}

	return s.emailService.SendVerifyCode(ctx, email, siteName)
}

// SendVerifyCodeAsync 寮傛鍙戦€侀偖绠遍獙璇佺爜骞惰繑鍥炲€掕鏃?
func (s *AuthService) SendVerifyCodeAsync(ctx context.Context, email string) (*SendVerifyCodeResult, error) {
	logger.LegacyPrintf("service.auth", "[Auth] SendVerifyCodeAsync called for email: %s", email)

	// 妫€鏌ユ槸鍚﹀紑鏀炬敞鍐岋紙榛樿鍏抽棴锛?
	if s.settingService == nil || !s.settingService.IsRegistrationEnabled(ctx) {
		logger.LegacyPrintf("service.auth", "%s", "[Auth] Registration is disabled")
		return nil, ErrRegDisabled
	}

	if isReservedEmail(email) {
		return nil, ErrEmailReserved
	}
	if err := s.validateRegistrationEmailPolicy(ctx, email); err != nil {
		return nil, err
	}

	// 妫€鏌ラ偖绠辨槸鍚﹀凡瀛樺湪
	existsEmail, err := s.userRepo.ExistsByEmail(ctx, email)
	if err != nil {
		logger.LegacyPrintf("service.auth", "[Auth] Database error checking email exists: %v", err)
		return nil, ErrServiceUnavailable
	}
	if existsEmail {
		logger.LegacyPrintf("service.auth", "[Auth] Email already exists: %s", email)
		return nil, ErrEmailExists
	}

	// 妫€鏌ラ偖浠堕槦鍒楁湇鍔℃槸鍚﹂厤缃?
	if s.emailQueueService == nil {
		logger.LegacyPrintf("service.auth", "%s", "[Auth] Email queue service not configured")
		return nil, errors.New("email queue service not configured")
	}

	// 鑾峰彇缃戠珯鍚嶇О
	siteName := "Sub2API"
	if s.settingService != nil {
		siteName = s.settingService.GetSiteName(ctx)
	}

	// 寮傛鍙戦€?
	logger.LegacyPrintf("service.auth", "[Auth] Enqueueing verify code for: %s", email)
	if err := s.emailQueueService.EnqueueVerifyCode(email, siteName); err != nil {
		logger.LegacyPrintf("service.auth", "[Auth] Failed to enqueue: %v", err)
		return nil, fmt.Errorf("enqueue verify code: %w", err)
	}

	logger.LegacyPrintf("service.auth", "[Auth] Verify code enqueued successfully for: %s", email)
	return &SendVerifyCodeResult{
		Countdown: 60, // 60绉掑€掕鏃?
	}, nil
}

// VerifyTurnstileForRegister 鍦ㄦ敞鍐屽満鏅笅楠岃瘉 Turnstile銆?
// 褰撻偖绠遍獙璇佸紑鍚笖宸叉彁浜ら獙璇佺爜鏃讹紝璇存槑楠岃瘉鐮佸彂閫侀樁娈靛凡瀹屾垚 Turnstile 鏍￠獙锛?
// 姝ゅ璺宠繃浜屾鏍￠獙锛岄伩鍏嶄竴娆℃€?token 鍦ㄦ敞鍐屾彁浜ゆ椂閲嶅浣跨敤瀵艰嚧璇姤澶辫触銆?
func (s *AuthService) VerifyTurnstileForRegister(ctx context.Context, token, remoteIP, verifyCode string) error {
	if s.IsEmailVerifyEnabled(ctx) && strings.TrimSpace(verifyCode) != "" {
		logger.LegacyPrintf("service.auth", "%s", "[Auth] Email verify flow detected, skip duplicate Turnstile check on register")
		return nil
	}
	return s.VerifyTurnstile(ctx, token, remoteIP)
}

// VerifyTurnstile 楠岃瘉Turnstile token
func (s *AuthService) VerifyTurnstile(ctx context.Context, token string, remoteIP string) error {
	required := s.cfg != nil && s.cfg.Server.Mode == "release" && s.cfg.Turnstile.Required

	if required {
		if s.settingService == nil {
			logger.LegacyPrintf("service.auth", "%s", "[Auth] Turnstile required but settings service is not configured")
			return ErrTurnstileNotConfigured
		}
		enabled := s.settingService.IsTurnstileEnabled(ctx)
		secretConfigured := s.settingService.GetTurnstileSecretKey(ctx) != ""
		if !enabled || !secretConfigured {
			logger.LegacyPrintf("service.auth", "[Auth] Turnstile required but not configured (enabled=%v, secret_configured=%v)", enabled, secretConfigured)
			return ErrTurnstileNotConfigured
		}
	}

	if s.turnstileService == nil {
		if required {
			logger.LegacyPrintf("service.auth", "%s", "[Auth] Turnstile required but service not configured")
			return ErrTurnstileNotConfigured
		}
		return nil // 鏈嶅姟鏈厤缃垯璺宠繃楠岃瘉
	}

	if !required && s.settingService != nil && s.settingService.IsTurnstileEnabled(ctx) && s.settingService.GetTurnstileSecretKey(ctx) == "" {
		logger.LegacyPrintf("service.auth", "%s", "[Auth] Turnstile enabled but secret key not configured")
	}

	return s.turnstileService.VerifyToken(ctx, token, remoteIP)
}

// IsTurnstileEnabled 妫€鏌ユ槸鍚﹀惎鐢═urnstile楠岃瘉
func (s *AuthService) IsTurnstileEnabled(ctx context.Context) bool {
	if s.turnstileService == nil {
		return false
	}
	return s.turnstileService.IsEnabled(ctx)
}

// IsRegistrationEnabled 妫€鏌ユ槸鍚﹀紑鏀炬敞鍐?
func (s *AuthService) IsRegistrationEnabled(ctx context.Context) bool {
	if s.settingService == nil {
		return false // 瀹夊叏榛樿锛歴ettingService 鏈厤缃椂鍏抽棴娉ㄥ唽
	}
	return s.settingService.IsRegistrationEnabled(ctx)
}

// IsEmailVerifyEnabled 妫€鏌ユ槸鍚﹀紑鍚偖浠堕獙璇?
func (s *AuthService) IsEmailVerifyEnabled(ctx context.Context) bool {
	if s.settingService == nil {
		return false
	}
	return s.settingService.IsEmailVerifyEnabled(ctx)
}

// Login 鐢ㄦ埛鐧诲綍锛岃繑鍥濲WT token
func (s *AuthService) Login(ctx context.Context, email, password string) (string, *User, error) {
	// 鏌ユ壘鐢ㄦ埛
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return "", nil, ErrInvalidCredentials
		}
		// 璁板綍鏁版嵁搴撻敊璇絾涓嶆毚闇茬粰鐢ㄦ埛
		logger.LegacyPrintf("service.auth", "[Auth] Database error during login: %v", err)
		return "", nil, ErrServiceUnavailable
	}

	// 楠岃瘉瀵嗙爜
	if !s.CheckPassword(password, user.PasswordHash) {
		return "", nil, ErrInvalidCredentials
	}

	// 妫€鏌ョ敤鎴风姸鎬?
	if !user.IsActive() {
		return "", nil, ErrUserNotActive
	}

	// 鐢熸垚JWT token
	token, err := s.GenerateToken(user)
	if err != nil {
		return "", nil, fmt.Errorf("generate token: %w", err)
	}

	return token, user, nil
}

// LoginOrRegisterOAuth 鐢ㄤ簬绗笁鏂?OAuth/SSO 鐧诲綍锛?
// - 濡傛灉閭宸插瓨鍦細鐩存帴鐧诲綍锛堜笉闇€瑕佹湰鍦板瘑鐮侊級
// - 濡傛灉閭涓嶅瓨鍦細鍒涘缓鏂扮敤鎴峰苟鐧诲綍
//
// 娉ㄦ剰锛氳鍑芥暟鐢ㄤ簬 LinuxDo OAuth 鐧诲綍鍦烘櫙锛堜笉鍚屼簬涓婃父璐﹀彿鐨?OAuth锛屼緥濡?Claude/OpenAI/Gemini锛夈€?
// 涓轰簡婊¤冻鐜版湁鏁版嵁搴撶害鏉燂紙闇€瑕佸瘑鐮佸搱甯岋級锛屾柊鐢ㄦ埛浼氱敓鎴愰殢鏈哄瘑鐮佸苟杩涜鍝堝笇淇濆瓨銆?
func (s *AuthService) LoginOrRegisterOAuth(ctx context.Context, email, username string) (string, *User, error) {
	email = strings.TrimSpace(email)
	if email == "" || len(email) > 255 {
		return "", nil, infraerrors.BadRequest("INVALID_EMAIL", "invalid email")
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return "", nil, infraerrors.BadRequest("INVALID_EMAIL", "invalid email")
	}

	username = strings.TrimSpace(username)
	if len([]rune(username)) > 100 {
		username = string([]rune(username)[:100])
	}

	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			// OAuth 棣栨鐧诲綍瑙嗕负娉ㄥ唽锛坒ail-close锛歴ettingService 鏈厤缃椂涓嶅厑璁告敞鍐岋級
			if s.settingService == nil || !s.settingService.IsRegistrationEnabled(ctx) {
				return "", nil, ErrRegDisabled
			}

			randomPassword, err := randomHexString(32)
			if err != nil {
				logger.LegacyPrintf("service.auth", "[Auth] Failed to generate random password for oauth signup: %v", err)
				return "", nil, ErrServiceUnavailable
			}
			hashedPassword, err := s.HashPassword(randomPassword)
			if err != nil {
				return "", nil, fmt.Errorf("hash password: %w", err)
			}

			// 鏂扮敤鎴烽粯璁ゅ€笺€?
			defaultBalance := s.cfg.Default.UserBalance
			defaultConcurrency := s.cfg.Default.UserConcurrency
			if s.settingService != nil {
				defaultBalance = s.settingService.GetDefaultBalance(ctx)
				defaultConcurrency = s.settingService.GetDefaultConcurrency(ctx)
			}

			newUser := &User{
				Email:        email,
				Username:     username,
				PasswordHash: hashedPassword,
				Role:         RoleUser,
				Balance:      defaultBalance,
				Concurrency:  defaultConcurrency,
				Status:       StatusActive,
			}

			if err := s.userRepo.Create(ctx, newUser); err != nil {
				if errors.Is(err, ErrEmailExists) {
					// 骞跺彂鍦烘櫙锛欸etByEmail 涓?Create 涔嬮棿鐢ㄦ埛琚垱寤恒€?
					user, err = s.userRepo.GetByEmail(ctx, email)
					if err != nil {
						logger.LegacyPrintf("service.auth", "[Auth] Database error getting user after conflict: %v", err)
						return "", nil, ErrServiceUnavailable
					}
				} else {
					logger.LegacyPrintf("service.auth", "[Auth] Database error creating oauth user: %v", err)
					return "", nil, ErrServiceUnavailable
				}
			} else {
				user = newUser
				s.assignDefaultSubscriptions(ctx, user.ID)
			}
		} else {
			logger.LegacyPrintf("service.auth", "[Auth] Database error during oauth login: %v", err)
			return "", nil, ErrServiceUnavailable
		}
	}

	if !user.IsActive() {
		return "", nil, ErrUserNotActive
	}

	// 灏藉姏琛ュ叏锛氬綋鐢ㄦ埛鍚嶄负绌烘椂锛屼娇鐢ㄧ涓夋柟杩斿洖鐨勭敤鎴峰悕鍥炲～銆?
	if user.Username == "" && username != "" {
		user.Username = username
		if err := s.userRepo.Update(ctx, user); err != nil {
			logger.LegacyPrintf("service.auth", "[Auth] Failed to update username after oauth login: %v", err)
		}
	}

	token, err := s.GenerateToken(user)
	if err != nil {
		return "", nil, fmt.Errorf("generate token: %w", err)
	}
	return token, user, nil
}

// LoginOrRegisterOAuthWithTokenPair 鐢ㄤ簬绗笁鏂?OAuth/SSO 鐧诲綍锛岃繑鍥炲畬鏁寸殑 TokenPair銆?
// 涓?LoginOrRegisterOAuth 鍔熻兘鐩稿悓锛屼絾杩斿洖 TokenPair 鑰岄潪鍗曚釜 token銆?
// invitationCode 浠呭湪閭€璇风爜娉ㄥ唽妯″紡涓嬫柊鐢ㄦ埛娉ㄥ唽鏃朵娇鐢紱宸叉湁璐﹀彿鐧诲綍鏃跺拷鐣ャ€?
func (s *AuthService) LoginOrRegisterOAuthWithTokenPair(ctx context.Context, email, username, invitationCode string) (*TokenPair, *User, error) {
	if s.refreshTokenCache == nil {
		return nil, nil, errors.New("refresh token cache not configured")
	}

	email = strings.TrimSpace(email)
	if email == "" || len(email) > 255 {
		return nil, nil, infraerrors.BadRequest("INVALID_EMAIL", "invalid email")
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return nil, nil, infraerrors.BadRequest("INVALID_EMAIL", "invalid email")
	}

	username = strings.TrimSpace(username)
	if len([]rune(username)) > 100 {
		username = string([]rune(username)[:100])
	}

	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			if s.settingService == nil || !s.settingService.IsRegistrationEnabled(ctx) {
				return nil, nil, ErrRegDisabled
			}

			var inviter *User
			if s.settingService != nil && s.settingService.IsInvitationCodeEnabled(ctx) {
				if strings.TrimSpace(invitationCode) == "" {
					return nil, nil, ErrOAuthInvitationRequired
				}
				resolvedInviter, err := s.lookupInviterByCode(ctx, invitationCode)
				if err != nil {
					logger.LegacyPrintf("service.auth", "[Auth] Invalid invitation code during oauth registration: code=%s err=%v", invitationCode, err)
					return nil, nil, ErrInvitationCodeInvalid
				}
				inviter = resolvedInviter
			}

			invitationReward := 0.0
			if inviter != nil && s.settingService != nil {
				invitationReward = s.settingService.GetInvitationReward(ctx)
			}

			randomPassword, err := randomHexString(32)
			if err != nil {
				logger.LegacyPrintf("service.auth", "[Auth] Failed to generate random password for oauth signup: %v", err)
				return nil, nil, ErrServiceUnavailable
			}
			hashedPassword, err := s.HashPassword(randomPassword)
			if err != nil {
				return nil, nil, fmt.Errorf("hash password: %w", err)
			}

			defaultBalance := s.cfg.Default.UserBalance
			defaultConcurrency := s.cfg.Default.UserConcurrency
			if s.settingService != nil {
				defaultBalance = s.settingService.GetDefaultBalance(ctx)
				defaultConcurrency = s.settingService.GetDefaultConcurrency(ctx)
			}

			newUser := &User{
				Email:        email,
				Username:     username,
				PasswordHash: hashedPassword,
				Role:         RoleUser,
				Balance:      defaultBalance,
				Concurrency:  defaultConcurrency,
				Status:       StatusActive,
			}

			if err := s.createUserWithInvitationReward(ctx, newUser, inviter, invitationReward); err != nil {
				if errors.Is(err, ErrEmailExists) {
					user, err = s.userRepo.GetByEmail(ctx, email)
					if err != nil {
						logger.LegacyPrintf("service.auth", "[Auth] Database error getting user after conflict: %v", err)
						return nil, nil, ErrServiceUnavailable
					}
				} else {
					logger.LegacyPrintf("service.auth", "[Auth] Database error creating oauth user: %v", err)
					return nil, nil, ErrServiceUnavailable
				}
			} else {
				user = newUser
			}
		} else {
			logger.LegacyPrintf("service.auth", "[Auth] Database error during oauth login: %v", err)
			return nil, nil, ErrServiceUnavailable
		}
	}

	if !user.IsActive() {
		return nil, nil, ErrUserNotActive
	}

	if user.Username == "" && username != "" {
		user.Username = username
		if err := s.userRepo.Update(ctx, user); err != nil {
			logger.LegacyPrintf("service.auth", "[Auth] Failed to update username after oauth login: %v", err)
		}
	}

	tokenPair, err := s.GenerateTokenPair(ctx, user, "")
	if err != nil {
		return nil, nil, fmt.Errorf("generate token pair: %w", err)
	}
	return tokenPair, user, nil
}

// pendingOAuthTokenTTL is the validity period for pending OAuth tokens.
const pendingOAuthTokenTTL = 10 * time.Minute

// pendingOAuthPurpose is the purpose claim value for pending OAuth registration tokens.
const pendingOAuthPurpose = "pending_oauth_registration"

type pendingOAuthClaims struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Purpose  string `json:"purpose"`
	jwt.RegisteredClaims
}

// CreatePendingOAuthToken generates a short-lived JWT that carries the OAuth identity
// while waiting for the user to supply an invitation code.
func (s *AuthService) CreatePendingOAuthToken(email, username string) (string, error) {
	now := time.Now()
	claims := &pendingOAuthClaims{
		Email:    email,
		Username: username,
		Purpose:  pendingOAuthPurpose,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(pendingOAuthTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.JWT.Secret))
}

// VerifyPendingOAuthToken validates a pending OAuth token and returns the embedded identity.
// Returns ErrInvalidToken when the token is invalid or expired.
func (s *AuthService) VerifyPendingOAuthToken(tokenStr string) (email, username string, err error) {
	if len(tokenStr) > maxTokenLength {
		return "", "", ErrInvalidToken
	}
	parser := jwt.NewParser(jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}))
	token, parseErr := parser.ParseWithClaims(tokenStr, &pendingOAuthClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(s.cfg.JWT.Secret), nil
	})
	if parseErr != nil {
		return "", "", ErrInvalidToken
	}
	claims, ok := token.Claims.(*pendingOAuthClaims)
	if !ok || !token.Valid {
		return "", "", ErrInvalidToken
	}
	if claims.Purpose != pendingOAuthPurpose {
		return "", "", ErrInvalidToken
	}
	return claims.Email, claims.Username, nil
}

func (s *AuthService) ValidateInvitationCode(ctx context.Context, invitationCode string) error {
	_, err := s.lookupInviterByCode(ctx, invitationCode)
	return err
}

func (s *AuthService) lookupInviterByCode(ctx context.Context, invitationCode string) (*User, error) {
	inviter, err := FindInviterByInviteCode(ctx, s.entClient, invitationCode)
	if err != nil {
		return nil, err
	}
	if inviter == nil || !inviter.IsActive() {
		return nil, ErrInvitationCodeInvalid
	}
	return inviter, nil
}

func inviterIDPointer(inviter *User) *int64 {
	if inviter == nil {
		return nil
	}
	id := inviter.ID
	return &id
}

func (s *AuthService) createUserWithInvitationReward(ctx context.Context, user *User, inviter *User, reward float64) error {
	if user == nil {
		return fmt.Errorf("user is required")
	}
	if user.InviteCode == "" {
		code, err := GenerateUniqueInviteCode(ctx, s.entClient)
		if err != nil {
			return fmt.Errorf("generate invite code: %w", err)
		}
		user.InviteCode = code
	}
	user.InvitedByID = inviterIDPointer(inviter)

	if s.entClient == nil {
		if err := s.userRepo.Create(ctx, user); err != nil {
			return err
		}
		if err := s.recordInvitationReward(ctx, inviter, reward, user.Email); err != nil {
			return err
		}
		s.assignDefaultSubscriptions(ctx, user.ID)
		s.invalidateInvitationRewardCaches(ctx, inviter, reward)
		return nil
	}

	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	txCtx := dbent.NewTxContext(ctx, tx)
	if err := s.userRepo.Create(txCtx, user); err != nil {
		return err
	}
	if err := s.recordInvitationReward(txCtx, inviter, reward, user.Email); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	s.assignDefaultSubscriptions(ctx, user.ID)
	s.invalidateInvitationRewardCaches(ctx, inviter, reward)
	return nil
}

func (s *AuthService) recordInvitationReward(ctx context.Context, inviter *User, reward float64, inviteeEmail string) error {
	if inviter == nil || inviter.ID <= 0 || reward <= 0 {
		return nil
	}
	if err := s.userRepo.UpdateBalance(ctx, inviter.ID, reward); err != nil {
		return fmt.Errorf("update inviter balance: %w", err)
	}
	if s.redeemRepo == nil {
		return nil
	}

	code, err := GenerateRedeemCode()
	if err != nil {
		return fmt.Errorf("generate invitation reward record code: %w", err)
	}
	usedBy := inviter.ID
	usedAt := time.Now()
	record := &RedeemCode{
		Code:   strings.ToUpper(code),
		Type:   AdjustmentTypeAdminBalance,
		Value:  reward,
		Status: StatusUsed,
		UsedBy: &usedBy,
		UsedAt: &usedAt,
		Notes:  fmt.Sprintf("invitation reward for %s", strings.TrimSpace(inviteeEmail)),
	}
	if err := s.redeemRepo.Create(ctx, record); err != nil {
		return fmt.Errorf("create invitation reward record: %w", err)
	}
	return nil
}

func (s *AuthService) invalidateInvitationRewardCaches(ctx context.Context, inviter *User, reward float64) {
	if inviter == nil || reward <= 0 || s.promoService == nil {
		return
	}
	s.promoService.InvalidateRewardCaches(ctx, inviter.ID, reward)
}
func (s *AuthService) assignDefaultSubscriptions(ctx context.Context, userID int64) {
	if s.settingService == nil || s.defaultSubAssigner == nil || userID <= 0 {
		return
	}
	items := s.settingService.GetDefaultSubscriptions(ctx)
	for _, item := range items {
		if _, _, err := s.defaultSubAssigner.AssignOrExtendSubscription(ctx, &AssignSubscriptionInput{
			UserID:       userID,
			GroupID:      item.GroupID,
			ValidityDays: item.ValidityDays,
			Notes:        "auto assigned by default user subscriptions setting",
		}); err != nil {
			logger.LegacyPrintf("service.auth", "[Auth] Failed to assign default subscription: user_id=%d group_id=%d err=%v", userID, item.GroupID, err)
		}
	}
}

func (s *AuthService) validateRegistrationEmailPolicy(ctx context.Context, email string) error {
	if s.settingService == nil {
		return nil
	}
	whitelist := s.settingService.GetRegistrationEmailSuffixWhitelist(ctx)
	if !IsRegistrationEmailSuffixAllowed(email, whitelist) {
		return buildEmailSuffixNotAllowedError(whitelist)
	}
	return nil
}

func buildEmailSuffixNotAllowedError(whitelist []string) error {
	if len(whitelist) == 0 {
		return ErrEmailSuffixNotAllowed
	}

	allowed := strings.Join(whitelist, ", ")
	return infraerrors.BadRequest(
		"EMAIL_SUFFIX_NOT_ALLOWED",
		fmt.Sprintf("email suffix is not allowed, allowed suffixes: %s", allowed),
	).WithMetadata(map[string]string{
		"allowed_suffixes":     strings.Join(whitelist, ","),
		"allowed_suffix_count": strconv.Itoa(len(whitelist)),
	})
}

// ValidateToken 楠岃瘉JWT token骞惰繑鍥炵敤鎴峰０鏄?
func (s *AuthService) ValidateToken(tokenString string) (*JWTClaims, error) {
	// 鍏堝仛闀垮害鏍￠獙锛屽敖鏃╂嫆缁濆紓甯歌秴闀?token锛岄檷浣?DoS 椋庨櫓銆?
	if len(tokenString) > maxTokenLength {
		return nil, ErrTokenTooLarge
	}

	// 浣跨敤瑙ｆ瀽鍣ㄥ苟闄愬埗鍙帴鍙楃殑绛惧悕绠楁硶锛岄槻姝㈢畻娉曟贩娣嗐€?
	parser := jwt.NewParser(jwt.WithValidMethods([]string{
		jwt.SigningMethodHS256.Name,
		jwt.SigningMethodHS384.Name,
		jwt.SigningMethodHS512.Name,
	}))

	// 淇濈暀榛樿 claims 鏍￠獙锛坋xp/nbf锛夛紝閬垮厤鏀捐杩囨湡鎴栨湭鐢熸晥鐨?token銆?
	token, err := parser.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (any, error) {
		// 楠岃瘉绛惧悕鏂规硶
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.cfg.JWT.Secret), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			// token 杩囨湡浣嗕粛杩斿洖 claims锛堢敤浜?RefreshToken 绛夊満鏅級
			// jwt-go 鍦ㄨВ鏋愭椂鍗充娇閬囧埌杩囨湡閿欒锛宼oken.Claims 浠嶄細琚～鍏?
			if claims, ok := token.Claims.(*JWTClaims); ok {
				return claims, ErrTokenExpired
			}
			return nil, ErrTokenExpired
		}
		return nil, ErrInvalidToken
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrInvalidToken
}

func randomHexString(byteLength int) (string, error) {
	if byteLength <= 0 {
		byteLength = 16
	}
	buf := make([]byte, byteLength)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func isReservedEmail(email string) bool {
	normalized := strings.ToLower(strings.TrimSpace(email))
	return strings.HasSuffix(normalized, LinuxDoConnectSyntheticEmailDomain) ||
		strings.HasSuffix(normalized, OIDCConnectSyntheticEmailDomain)
}

// GenerateToken 鐢熸垚JWT access token
// 浣跨敤鏂扮殑access_token_expire_minutes閰嶇疆椤癸紙濡傛灉閰嶇疆浜嗭級锛屽惁鍒欏洖閫€鍒癳xpire_hour
func (s *AuthService) GenerateToken(user *User) (string, error) {
	now := time.Now()
	var expiresAt time.Time
	if s.cfg.JWT.AccessTokenExpireMinutes > 0 {
		expiresAt = now.Add(time.Duration(s.cfg.JWT.AccessTokenExpireMinutes) * time.Minute)
	} else {
		// 鍚戝悗鍏煎锛氫娇鐢ㄦ棫鐨別xpire_hour閰嶇疆
		expiresAt = now.Add(time.Duration(s.cfg.JWT.ExpireHour) * time.Hour)
	}

	claims := &JWTClaims{
		UserID:       user.ID,
		Email:        user.Email,
		Role:         user.Role,
		TokenVersion: user.TokenVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.cfg.JWT.Secret))
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}

	return tokenString, nil
}

// GetAccessTokenExpiresIn 杩斿洖Access Token鐨勬湁鏁堟湡锛堢锛?
// 鐢ㄤ簬鍓嶇璁剧疆鍒锋柊瀹氭椂鍣?
func (s *AuthService) GetAccessTokenExpiresIn() int {
	if s.cfg.JWT.AccessTokenExpireMinutes > 0 {
		return s.cfg.JWT.AccessTokenExpireMinutes * 60
	}
	return s.cfg.JWT.ExpireHour * 3600
}

// HashPassword 浣跨敤bcrypt鍔犲瘑瀵嗙爜
func (s *AuthService) HashPassword(password string) (string, error) {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedBytes), nil
}

// CheckPassword 楠岃瘉瀵嗙爜鏄惁鍖归厤
func (s *AuthService) CheckPassword(password, hashedPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}

// RefreshToken 鍒锋柊token
func (s *AuthService) RefreshToken(ctx context.Context, oldTokenString string) (string, error) {
	// 楠岃瘉鏃oken锛堝嵆浣胯繃鏈熶篃鍏佽锛岀敤浜庡埛鏂帮級
	claims, err := s.ValidateToken(oldTokenString)
	if err != nil && !errors.Is(err, ErrTokenExpired) {
		return "", err
	}

	// 鑾峰彇鏈€鏂扮殑鐢ㄦ埛淇℃伅
	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return "", ErrInvalidToken
		}
		logger.LegacyPrintf("service.auth", "[Auth] Database error refreshing token: %v", err)
		return "", ErrServiceUnavailable
	}

	// 妫€鏌ョ敤鎴风姸鎬?
	if !user.IsActive() {
		return "", ErrUserNotActive
	}

	// Security: Check TokenVersion to prevent refreshing revoked tokens
	// This ensures tokens issued before a password change cannot be refreshed
	if claims.TokenVersion != user.TokenVersion {
		return "", ErrTokenRevoked
	}

	// 鐢熸垚鏂皌oken
	return s.GenerateToken(user)
}

// IsPasswordResetEnabled 妫€鏌ユ槸鍚﹀惎鐢ㄥ瘑鐮侀噸缃姛鑳?
// 瑕佹眰锛氬繀椤诲悓鏃跺紑鍚偖浠堕獙璇佷笖 SMTP 閰嶇疆姝ｇ‘
func (s *AuthService) IsPasswordResetEnabled(ctx context.Context) bool {
	if s.settingService == nil {
		return false
	}
	// Must have email verification enabled and SMTP configured
	if !s.settingService.IsEmailVerifyEnabled(ctx) {
		return false
	}
	return s.settingService.IsPasswordResetEnabled(ctx)
}

// preparePasswordReset validates the password reset request and returns necessary data
// Returns (siteName, resetURL, shouldProceed)
// shouldProceed is false when we should silently return success (to prevent enumeration)
func (s *AuthService) preparePasswordReset(ctx context.Context, email, frontendBaseURL string) (string, string, bool) {
	// Check if user exists (but don't reveal this to the caller)
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			// Security: Log but don't reveal that user doesn't exist
			logger.LegacyPrintf("service.auth", "[Auth] Password reset requested for non-existent email: %s", email)
			return "", "", false
		}
		logger.LegacyPrintf("service.auth", "[Auth] Database error checking email for password reset: %v", err)
		return "", "", false
	}

	// Check if user is active
	if !user.IsActive() {
		logger.LegacyPrintf("service.auth", "[Auth] Password reset requested for inactive user: %s", email)
		return "", "", false
	}

	// Get site name
	siteName := "Sub2API"
	if s.settingService != nil {
		siteName = s.settingService.GetSiteName(ctx)
	}

	// Build reset URL base
	resetURL := fmt.Sprintf("%s/reset-password", strings.TrimSuffix(frontendBaseURL, "/"))

	return siteName, resetURL, true
}

// RequestPasswordReset 璇锋眰瀵嗙爜閲嶇疆锛堝悓姝ュ彂閫侊級
// Security: Returns the same response regardless of whether the email exists (prevent user enumeration)
func (s *AuthService) RequestPasswordReset(ctx context.Context, email, frontendBaseURL string) error {
	if !s.IsPasswordResetEnabled(ctx) {
		return infraerrors.Forbidden("PASSWORD_RESET_DISABLED", "password reset is not enabled")
	}
	if s.emailService == nil {
		return ErrServiceUnavailable
	}

	siteName, resetURL, shouldProceed := s.preparePasswordReset(ctx, email, frontendBaseURL)
	if !shouldProceed {
		return nil // Silent success to prevent enumeration
	}

	if err := s.emailService.SendPasswordResetEmail(ctx, email, siteName, resetURL); err != nil {
		logger.LegacyPrintf("service.auth", "[Auth] Failed to send password reset email to %s: %v", email, err)
		return nil // Silent success to prevent enumeration
	}

	logger.LegacyPrintf("service.auth", "[Auth] Password reset email sent to: %s", email)
	return nil
}

// RequestPasswordResetAsync 寮傛璇锋眰瀵嗙爜閲嶇疆锛堥槦鍒楀彂閫侊級
// Security: Returns the same response regardless of whether the email exists (prevent user enumeration)
func (s *AuthService) RequestPasswordResetAsync(ctx context.Context, email, frontendBaseURL string) error {
	if !s.IsPasswordResetEnabled(ctx) {
		return infraerrors.Forbidden("PASSWORD_RESET_DISABLED", "password reset is not enabled")
	}
	if s.emailQueueService == nil {
		return ErrServiceUnavailable
	}

	siteName, resetURL, shouldProceed := s.preparePasswordReset(ctx, email, frontendBaseURL)
	if !shouldProceed {
		return nil // Silent success to prevent enumeration
	}

	if err := s.emailQueueService.EnqueuePasswordReset(email, siteName, resetURL); err != nil {
		logger.LegacyPrintf("service.auth", "[Auth] Failed to enqueue password reset email for %s: %v", email, err)
		return nil // Silent success to prevent enumeration
	}

	logger.LegacyPrintf("service.auth", "[Auth] Password reset email enqueued for: %s", email)
	return nil
}

// ResetPassword 閲嶇疆瀵嗙爜
// Security: Increments TokenVersion to invalidate all existing JWT tokens
func (s *AuthService) ResetPassword(ctx context.Context, email, token, newPassword string) error {
	// Check if password reset is enabled
	if !s.IsPasswordResetEnabled(ctx) {
		return infraerrors.Forbidden("PASSWORD_RESET_DISABLED", "password reset is not enabled")
	}

	if s.emailService == nil {
		return ErrServiceUnavailable
	}

	// Verify and consume the reset token (one-time use)
	if err := s.emailService.ConsumePasswordResetToken(ctx, email, token); err != nil {
		return err
	}

	// Get user
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return ErrInvalidResetToken // Token was valid but user was deleted
		}
		logger.LegacyPrintf("service.auth", "[Auth] Database error getting user for password reset: %v", err)
		return ErrServiceUnavailable
	}

	// Check if user is active
	if !user.IsActive() {
		return ErrUserNotActive
	}

	// Hash new password
	hashedPassword, err := s.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	// Update password and increment TokenVersion
	user.PasswordHash = hashedPassword
	user.TokenVersion++ // Invalidate all existing tokens

	if err := s.userRepo.Update(ctx, user); err != nil {
		logger.LegacyPrintf("service.auth", "[Auth] Database error updating password for user %d: %v", user.ID, err)
		return ErrServiceUnavailable
	}

	// Also revoke all refresh tokens for this user
	if err := s.RevokeAllUserSessions(ctx, user.ID); err != nil {
		logger.LegacyPrintf("service.auth", "[Auth] Failed to revoke refresh tokens for user %d: %v", user.ID, err)
		// Don't return error - password was already changed successfully
	}

	logger.LegacyPrintf("service.auth", "[Auth] Password reset successful for user: %s", email)
	return nil
}

// ==================== Refresh Token Methods ====================

// TokenPair 鍖呭惈Access Token鍜孯efresh Token
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"` // Access Token鏈夋晥鏈燂紙绉掞級
}

// TokenPairWithUser extends TokenPair with user role for backend mode checks
type TokenPairWithUser struct {
	TokenPair
	UserRole string
}

// GenerateTokenPair 鐢熸垚Access Token鍜孯efresh Token瀵?
// familyID: 鍙€夌殑Token瀹舵棌ID锛岀敤浜嶵oken杞浆鏃朵繚鎸佸鏃忓叧绯?
func (s *AuthService) GenerateTokenPair(ctx context.Context, user *User, familyID string) (*TokenPair, error) {
	// 妫€鏌?refreshTokenCache 鏄惁鍙敤
	if s.refreshTokenCache == nil {
		return nil, errors.New("refresh token cache not configured")
	}

	// 鐢熸垚Access Token
	accessToken, err := s.GenerateToken(user)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	// 鐢熸垚Refresh Token
	refreshToken, err := s.generateRefreshToken(ctx, user, familyID)
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    s.GetAccessTokenExpiresIn(),
	}, nil
}

// generateRefreshToken 鐢熸垚骞跺瓨鍌≧efresh Token
func (s *AuthService) generateRefreshToken(ctx context.Context, user *User, familyID string) (string, error) {
	// 鐢熸垚闅忔満Token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("generate random bytes: %w", err)
	}
	rawToken := refreshTokenPrefix + hex.EncodeToString(tokenBytes)

	// 璁＄畻Token鍝堝笇锛堝瓨鍌ㄥ搱甯岃€岄潪鍘熷Token锛?
	tokenHash := hashToken(rawToken)

	// 濡傛灉娌℃湁鎻愪緵familyID锛岀敓鎴愭柊鐨?
	if familyID == "" {
		familyBytes := make([]byte, 16)
		if _, err := rand.Read(familyBytes); err != nil {
			return "", fmt.Errorf("generate family id: %w", err)
		}
		familyID = hex.EncodeToString(familyBytes)
	}

	now := time.Now()
	ttl := time.Duration(s.cfg.JWT.RefreshTokenExpireDays) * 24 * time.Hour

	data := &RefreshTokenData{
		UserID:       user.ID,
		TokenVersion: user.TokenVersion,
		FamilyID:     familyID,
		CreatedAt:    now,
		ExpiresAt:    now.Add(ttl),
	}

	// 瀛樺偍Token鏁版嵁
	if err := s.refreshTokenCache.StoreRefreshToken(ctx, tokenHash, data, ttl); err != nil {
		return "", fmt.Errorf("store refresh token: %w", err)
	}

	// 娣诲姞鍒扮敤鎴稵oken闆嗗悎
	if err := s.refreshTokenCache.AddToUserTokenSet(ctx, user.ID, tokenHash, ttl); err != nil {
		logger.LegacyPrintf("service.auth", "[Auth] Failed to add token to user set: %v", err)
		// 涓嶅奖鍝嶄富娴佺▼
	}

	// 娣诲姞鍒板鏃廡oken闆嗗悎
	if err := s.refreshTokenCache.AddToFamilyTokenSet(ctx, familyID, tokenHash, ttl); err != nil {
		logger.LegacyPrintf("service.auth", "[Auth] Failed to add token to family set: %v", err)
		// 涓嶅奖鍝嶄富娴佺▼
	}

	return rawToken, nil
}

// RefreshTokenPair 浣跨敤Refresh Token鍒锋柊Token瀵?
// 瀹炵幇Token杞浆锛氭瘡娆″埛鏂伴兘浼氱敓鎴愭柊鐨凴efresh Token锛屾棫Token绔嬪嵆澶辨晥
func (s *AuthService) RefreshTokenPair(ctx context.Context, refreshToken string) (*TokenPairWithUser, error) {
	// 妫€鏌?refreshTokenCache 鏄惁鍙敤
	if s.refreshTokenCache == nil {
		return nil, ErrRefreshTokenInvalid
	}

	// 楠岃瘉Token鏍煎紡
	if !strings.HasPrefix(refreshToken, refreshTokenPrefix) {
		return nil, ErrRefreshTokenInvalid
	}

	tokenHash := hashToken(refreshToken)

	// 鑾峰彇Token鏁版嵁
	data, err := s.refreshTokenCache.GetRefreshToken(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, ErrRefreshTokenNotFound) {
			// Token涓嶅瓨鍦紝鍙兘鏄凡琚娇鐢紙Token杞浆锛夋垨宸茶繃鏈?
			logger.LegacyPrintf("service.auth", "[Auth] Refresh token not found, possible reuse attack")
			return nil, ErrRefreshTokenInvalid
		}
		logger.LegacyPrintf("service.auth", "[Auth] Error getting refresh token: %v", err)
		return nil, ErrServiceUnavailable
	}

	// 妫€鏌oken鏄惁杩囨湡
	if time.Now().After(data.ExpiresAt) {
		// 鍒犻櫎杩囨湡Token
		_ = s.refreshTokenCache.DeleteRefreshToken(ctx, tokenHash)
		return nil, ErrRefreshTokenExpired
	}

	// 鑾峰彇鐢ㄦ埛淇℃伅
	user, err := s.userRepo.GetByID(ctx, data.UserID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			// 鐢ㄦ埛宸插垹闄わ紝鎾ら攢鏁翠釜Token瀹舵棌
			_ = s.refreshTokenCache.DeleteTokenFamily(ctx, data.FamilyID)
			return nil, ErrRefreshTokenInvalid
		}
		logger.LegacyPrintf("service.auth", "[Auth] Database error getting user for token refresh: %v", err)
		return nil, ErrServiceUnavailable
	}

	// 妫€鏌ョ敤鎴风姸鎬?
	if !user.IsActive() {
		// 鐢ㄦ埛琚鐢紝鎾ら攢鏁翠釜Token瀹舵棌
		_ = s.refreshTokenCache.DeleteTokenFamily(ctx, data.FamilyID)
		return nil, ErrUserNotActive
	}

	// 妫€鏌okenVersion锛堝瘑鐮佹洿鏀瑰悗鎵€鏈塗oken澶辨晥锛?
	if data.TokenVersion != user.TokenVersion {
		// TokenVersion涓嶅尮閰嶏紝鎾ら攢鏁翠釜Token瀹舵棌
		_ = s.refreshTokenCache.DeleteTokenFamily(ctx, data.FamilyID)
		return nil, ErrTokenRevoked
	}

	// Token杞浆锛氱珛鍗充娇鏃oken澶辨晥
	if err := s.refreshTokenCache.DeleteRefreshToken(ctx, tokenHash); err != nil {
		logger.LegacyPrintf("service.auth", "[Auth] Failed to delete old refresh token: %v", err)
		// 缁х画澶勭悊锛屼笉褰卞搷涓绘祦绋?
	}

	// 鐢熸垚鏂扮殑Token瀵癸紝淇濇寔鍚屼竴涓鏃廔D
	pair, err := s.GenerateTokenPair(ctx, user, data.FamilyID)
	if err != nil {
		return nil, err
	}
	return &TokenPairWithUser{
		TokenPair: *pair,
		UserRole:  user.Role,
	}, nil
}

// RevokeRefreshToken 鎾ら攢鍗曚釜Refresh Token
func (s *AuthService) RevokeRefreshToken(ctx context.Context, refreshToken string) error {
	if s.refreshTokenCache == nil {
		return nil // No-op if cache not configured
	}
	if !strings.HasPrefix(refreshToken, refreshTokenPrefix) {
		return ErrRefreshTokenInvalid
	}

	tokenHash := hashToken(refreshToken)
	return s.refreshTokenCache.DeleteRefreshToken(ctx, tokenHash)
}

// RevokeAllUserSessions 鎾ら攢鐢ㄦ埛鐨勬墍鏈変細璇濓紙鎵€鏈塕efresh Token锛?
// 鐢ㄤ簬瀵嗙爜鏇存敼鎴栫敤鎴蜂富鍔ㄧ櫥鍑烘墍鏈夎澶?
func (s *AuthService) RevokeAllUserSessions(ctx context.Context, userID int64) error {
	if s.refreshTokenCache == nil {
		return nil // No-op if cache not configured
	}
	return s.refreshTokenCache.DeleteUserRefreshTokens(ctx, userID)
}

// hashToken 璁＄畻Token鐨凷HA256鍝堝笇
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
