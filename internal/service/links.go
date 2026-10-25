package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/bcrowe306/linksy/internal/auth"
	"github.com/bcrowe306/linksy/internal/models"
)

var (
	ErrLinkPaused  = errors.New("this link is paused")
	ErrLinkExpired = errors.New("this link has expired")
)

const (
	minLinkPassword = 4
	maxUTMLen       = 100
	unlockTTL       = 12 * time.Hour
)

// UTM carries campaign parameters appended to a link's destination at redirect time.
type UTM struct {
	Source   string `json:"utm_source"`
	Medium   string `json:"utm_medium"`
	Campaign string `json:"utm_campaign"`
	Term     string `json:"utm_term"`
	Content  string `json:"utm_content"`
}

func (u UTM) clean() (UTM, error) {
	fields := []*string{&u.Source, &u.Medium, &u.Campaign, &u.Term, &u.Content}
	for _, f := range fields {
		*f = strings.TrimSpace(*f)
		if len(*f) > maxUTMLen {
			return u, fmt.Errorf("UTM values must be %d characters or fewer", maxUTMLen)
		}
	}
	return u, nil
}

func applyUTM(l *models.Link, u UTM) {
	l.UTMSource, l.UTMMedium, l.UTMCampaign, l.UTMTerm, l.UTMContent = u.Source, u.Medium, u.Campaign, u.Term, u.Content
}

// Destination returns the link's target with UTM parameters added. Parameters
// already present on the target URL win, so explicit tagging is never overwritten.
func Destination(l *models.Link) string {
	u, err := url.Parse(l.TargetURL)
	if err != nil {
		return l.TargetURL
	}
	q := u.Query()
	added := false
	for k, v := range map[string]string{
		"utm_source": l.UTMSource, "utm_medium": l.UTMMedium, "utm_campaign": l.UTMCampaign,
		"utm_term": l.UTMTerm, "utm_content": l.UTMContent,
	} {
		if v != "" && q.Get(k) == "" {
			q.Set(k, v)
			added = true
		}
	}
	if !added {
		return l.TargetURL
	}
	u.RawQuery = q.Encode()
	return u.String()
}

func hashLinkPassword(pw string) (string, error) {
	if len(pw) < minLinkPassword {
		return "", fmt.Errorf("link passwords must be at least %d characters", minLinkPassword)
	}
	if len(pw) > 72 {
		return "", errors.New("link passwords must be 72 characters or fewer")
	}
	h, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	return string(h), err
}

func validateFallback(u string) (string, error) {
	u = strings.TrimSpace(u)
	if u == "" {
		return "", nil
	}
	if err := ValidateTargetURL(u); err != nil {
		return "", fmt.Errorf("fallback URL: %w", err)
	}
	return u, nil
}

// ----------------- Links Management -----------------

type CreateLinkInput struct {
	TargetURL   string     `json:"target_url"`
	CustomSlug  string     `json:"custom_slug"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	ExpiresAt   *time.Time `json:"expires_at"`
	Password    string     `json:"password"`
	UTM         UTM        `json:"utm"`
	FallbackURL string     `json:"fallback_url"`
}

func (s *Service) CreateShortLink(a Actor, input CreateLinkInput) (*models.Link, error) {
	if err := a.require(auth.PermEditAssets); err != nil {
		return nil, err
	}
	if err := ValidateTargetURL(input.TargetURL); err != nil {
		return nil, err
	}
	utm, err := input.UTM.clean()
	if err != nil {
		return nil, err
	}
	fallback, err := validateFallback(input.FallbackURL)
	if err != nil {
		return nil, err
	}
	slug, err := s.allocateSlug(input.CustomSlug)
	if err != nil {
		return nil, err
	}

	title := strings.TrimSpace(input.Title)
	if title == "" {
		title = defaultLinkTitle(input.TargetURL)
	}

	link := &models.Link{
		TeamID:      a.TeamID,
		UserID:      a.UserID,
		Slug:        slug,
		TargetURL:   strings.TrimSpace(input.TargetURL),
		Title:       title,
		Description: input.Description,
		IsActive:    true,
		ExpiresAt:   input.ExpiresAt,
		FallbackURL: fallback,
	}
	applyUTM(link, utm)
	if input.Password != "" {
		if link.PasswordHash, err = hashLinkPassword(input.Password); err != nil {
			return nil, err
		}
	}

	if err := s.repo.CreateLink(link); err != nil {
		return nil, fmt.Errorf("failed to save link: %w", err)
	}
	s.audit(a, "link.create", "link", link.ID, map[string]any{"slug": link.Slug})
	return link, nil
}

// defaultLinkTitle uses the destination's host and path ("example.com/spring").
func defaultLinkTitle(target string) string {
	u, err := url.Parse(strings.TrimSpace(target))
	if err != nil || u.Host == "" {
		return target
	}
	t := strings.TrimPrefix(u.Host, "www.") + strings.TrimSuffix(u.Path, "/")
	if len(t) > 80 {
		t = t[:80]
	}
	return t
}

// LinkState describes why a link can't redirect right now.
type LinkState int

const (
	LinkOK LinkState = iota
	LinkPaused
	LinkExpired
)

// LookupLink finds a link by slug for the public redirector and reports its state.
func (s *Service) LookupLink(slug string) (*models.Link, LinkState, error) {
	link, err := s.repo.GetLinkBySlug(slug)
	if err != nil {
		return nil, LinkOK, ErrLinkNotFound
	}
	if !link.IsActive {
		return link, LinkPaused, nil
	}
	if link.ExpiresAt != nil && link.ExpiresAt.Before(time.Now().UTC()) {
		return link, LinkExpired, nil
	}
	return link, LinkOK, nil
}

// ResolveShortLink returns an active, unexpired link and counts the click.
func (s *Service) ResolveShortLink(slug string) (*models.Link, error) {
	link, state, err := s.LookupLink(slug)
	if err != nil || state != LinkOK {
		return nil, ErrLinkNotFound
	}
	s.CountClick(link.ID)
	return link, nil
}

func (s *Service) CountClick(linkID string) { _ = s.repo.IncrementLinkClicks(linkID) }

// CheckLinkPassword verifies a visitor-supplied password.
func (s *Service) CheckLinkPassword(link *models.Link, pw string) bool {
	return link.PasswordHash != "" && bcrypt.CompareHashAndPassword([]byte(link.PasswordHash), []byte(pw)) == nil
}

// UnlockToken returns a signed token proving the visitor entered the password.
// It embeds a fingerprint of the current hash, so changing the password revokes it.
func (s *Service) UnlockToken(link *models.Link, now time.Time) string {
	exp := strconv.FormatInt(now.Add(unlockTTL).Unix(), 10)
	return exp + "." + s.unlockSig(link, exp)
}

func (s *Service) unlockSig(link *models.Link, exp string) string {
	mac := hmac.New(sha256.New, []byte(s.ipSalt+"|link-unlock"))
	mac.Write([]byte(link.ID + "|" + link.PasswordHash + "|" + exp))
	return hex.EncodeToString(mac.Sum(nil))[:32]
}

// ValidUnlock checks a token from UnlockToken.
func (s *Service) ValidUnlock(link *models.Link, token string, now time.Time) bool {
	exp, sig, ok := strings.Cut(token, ".")
	if !ok {
		return false
	}
	n, err := strconv.ParseInt(exp, 10, 64)
	if err != nil || now.Unix() > n {
		return false
	}
	return hmac.Equal([]byte(sig), []byte(s.unlockSig(link, exp)))
}

type UpdateLinkInput struct {
	TargetURL   string
	Slug        string // empty = unchanged
	Title       string
	Description string
	IsActive    bool
	ExpiresAt   *time.Time
	Password    *string // nil = unchanged, "" = remove protection
	UTM         *UTM    // nil = unchanged
	FallbackURL *string // nil = unchanged
}

func (s *Service) UpdateShortLink(a Actor, id string, in UpdateLinkInput) (*models.Link, error) {
	if err := a.require(auth.PermEditAssets); err != nil {
		return nil, err
	}
	link, err := s.repo.GetLink(id, a.TeamID)
	if err != nil {
		return nil, ErrLinkNotFound
	}

	if in.TargetURL != "" && in.TargetURL != link.TargetURL {
		if err := ValidateTargetURL(in.TargetURL); err != nil {
			return nil, err
		}
		link.TargetURL = strings.TrimSpace(in.TargetURL)
	}
	meta := map[string]any{"slug": link.Slug}
	if slug := strings.TrimSpace(in.Slug); slug != "" && slug != link.Slug {
		newSlug, err := s.allocateSlug(slug)
		if err != nil {
			return nil, err
		}
		meta["previous_slug"] = link.Slug
		link.Slug = newSlug
	}
	if in.Title != "" {
		link.Title = in.Title
	}
	link.Description = in.Description
	link.IsActive = in.IsActive
	link.ExpiresAt = in.ExpiresAt
	if in.Password != nil {
		if *in.Password == "" {
			link.PasswordHash = ""
		} else if link.PasswordHash, err = hashLinkPassword(*in.Password); err != nil {
			return nil, err
		}
		meta["password_changed"] = true
	}
	if in.UTM != nil {
		utm, err := in.UTM.clean()
		if err != nil {
			return nil, err
		}
		applyUTM(link, utm)
	}
	if in.FallbackURL != nil {
		if link.FallbackURL, err = validateFallback(*in.FallbackURL); err != nil {
			return nil, err
		}
	}

	if err := s.repo.UpdateLink(link); err != nil {
		return nil, err
	}
	s.audit(a, "link.update", "link", link.ID, meta)
	return link, nil
}

func (s *Service) ListLinks(a Actor, limit, offset int) ([]models.Link, error) {
	if err := a.require(auth.PermViewAssets); err != nil {
		return nil, err
	}
	return s.repo.ListLinks(a.TeamID, clampLimit(limit), max(offset, 0))
}

func (s *Service) GetLink(a Actor, id string) (*models.Link, error) {
	if err := a.require(auth.PermViewAssets); err != nil {
		return nil, err
	}
	return s.repo.GetLink(id, a.TeamID)
}

func (s *Service) DeleteLink(a Actor, id string) error {
	if err := a.require(auth.PermEditAssets); err != nil {
		return err
	}
	if err := s.repo.DeleteLink(id, a.TeamID); err != nil {
		return err
	}
	s.audit(a, "link.delete", "link", id, nil)
	return nil
}
