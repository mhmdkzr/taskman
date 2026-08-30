package loginui

import (
	"context"
	"errors"
	"fmt"
)

// Register creates a new ZITADEL user and emails a verification link. It
// returns the ZITADEL user ID: not a secret (verifying still requires the
// emailed code), and needed by the frontend to complete verification and
// optional TOTP enrollment afterward.
func (s *Service) Register(
	ctx context.Context, loginName, email, password, givenName, familyName string,
) (string, error) {
	userID, emailCode, err := s.adminAPI.registerHuman(ctx, loginName, email, password, givenName, familyName)
	if err != nil {
		return "", err
	}
	link := fmt.Sprintf("%s/verify-email?userId=%s&code=%s", s.frontendBaseURL, userID, emailCode)
	body := fmt.Sprintf(
		"Welcome! Verify your email by visiting the link below.\n\n%s\n\n"+
			"If you didn't create this account, you can ignore this email.",
		link,
	)
	if err := s.mailer.Send(ctx, email, "Verify your email", body); err != nil {
		return "", fmt.Errorf("send verification email: %w", err)
	}
	return userID, nil
}

// VerifyEmail confirms a registration's email-verification code.
func (s *Service) VerifyEmail(ctx context.Context, userID, code string) error {
	return s.adminAPI.verifyEmail(ctx, userID, code)
}

// RequestPasswordReset emails a password-reset link for loginName if a
// matching account exists. It succeeds either way, so the caller can't use
// it to enumerate accounts.
func (s *Service) RequestPasswordReset(ctx context.Context, loginName string) error {
	userID, email, err := s.adminAPI.findUserByLoginName(ctx, loginName)
	if err != nil {
		if errors.Is(err, errInvalidCredentials) {
			return nil // unknown login name: succeed anyway to avoid leaking account existence
		}
		return err
	}
	code, err := s.adminAPI.requestPasswordReset(ctx, userID)
	if err != nil {
		return err
	}
	link := fmt.Sprintf("%s/reset-password?userId=%s&code=%s", s.frontendBaseURL, userID, code)
	body := fmt.Sprintf(
		"Reset your password by visiting the link below.\n\n%s\n\n"+
			"If you didn't request this, you can ignore this email.",
		link,
	)
	if err := s.mailer.Send(ctx, email, "Reset your password", body); err != nil {
		return fmt.Errorf("send password reset email: %w", err)
	}
	return nil
}

// ResetPassword sets a new password for userID using the code from
// RequestPasswordReset's email.
func (s *Service) ResetPassword(ctx context.Context, userID, code, newPassword string) error {
	return s.adminAPI.setPassword(ctx, userID, newPassword, code)
}

// StartTOTPEnrollment begins TOTP enrollment for userID (called right after
// registration, as an optional "enable 2FA now" step), returning the
// otpauth: URI and raw secret for the user's authenticator app.
func (s *Service) StartTOTPEnrollment(ctx context.Context, userID string) (string, string, error) {
	return s.adminAPI.registerTOTP(ctx, userID)
}

// ConfirmTOTPEnrollment completes TOTP enrollment for userID, verifying the
// caller controls the enrolled authenticator.
func (s *Service) ConfirmTOTPEnrollment(ctx context.Context, userID, code string) error {
	return s.adminAPI.verifyTOTPRegistration(ctx, userID, code)
}
