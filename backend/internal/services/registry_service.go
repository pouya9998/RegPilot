package services

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/regpilot/regpilot/internal/config"
	"github.com/regpilot/regpilot/internal/models"
	"github.com/regpilot/regpilot/internal/registry"
	"github.com/regpilot/regpilot/internal/repositories"
	"github.com/regpilot/regpilot/internal/utils"
	"gorm.io/gorm"
)

type RegistryService struct {
	regs      *repositories.RegistryRepository
	audit     *repositories.AuditRepository
	retention *repositories.RetentionRepository
	cfg       *config.Config
}

func NewRegistryService(
	regs *repositories.RegistryRepository,
	audit *repositories.AuditRepository,
	retention *repositories.RetentionRepository,
	cfg *config.Config,
) *RegistryService {
	return &RegistryService{regs: regs, audit: audit, retention: retention, cfg: cfg}
}

type CreateRegistryInput struct {
	Name               string `json:"name" binding:"required,min=1,max=128"`
	URL                string `json:"url" binding:"required"`
	Username           string `json:"username"`
	Password           string `json:"password"`
	InsecureSkipVerify bool   `json:"insecure_skip_verify"`
}

type UpdateRegistryInput struct {
	Name               *string `json:"name"`
	URL                *string `json:"url"`
	Username           *string `json:"username"`
	Password           *string `json:"password"`
	InsecureSkipVerify *bool   `json:"insecure_skip_verify"`
}

func (s *RegistryService) clientFor(reg *models.Registry) (registry.Client, error) {
	pass, err := utils.Decrypt(reg.PasswordEncrypted, s.cfg.EncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("decrypt credentials: %w", err)
	}
	return registry.NewClient(registry.ProviderDockerV2, registry.Config{
		BaseURL:            reg.URL,
		Username:           reg.Username,
		Password:           pass,
		InsecureSkipVerify: reg.InsecureSkipVerify,
	})
}

func validateRegistryURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return errors.New("url must be a valid http(s) URL")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return errors.New("url scheme must be http or https")
	}
	return nil
}

func (s *RegistryService) List() ([]models.RegistryResponse, error) {
	list, err := s.regs.List()
	if err != nil {
		return nil, err
	}
	out := make([]models.RegistryResponse, 0, len(list))
	for i := range list {
		out = append(out, list[i].ToResponse())
	}
	return out, nil
}

func (s *RegistryService) Get(id uint) (*models.RegistryResponse, error) {
	reg, err := s.regs.FindByID(id)
	if err != nil {
		return nil, err
	}
	resp := reg.ToResponse()
	return &resp, nil
}

func (s *RegistryService) Create(actorID uint, actorName string, in CreateRegistryInput) (*models.RegistryResponse, error) {
	in.URL = strings.TrimRight(strings.TrimSpace(in.URL), "/")
	if err := validateRegistryURL(in.URL); err != nil {
		return nil, err
	}
	enc, err := utils.Encrypt(in.Password, s.cfg.EncryptionKey)
	if err != nil {
		return nil, err
	}
	reg := &models.Registry{
		Name:               strings.TrimSpace(in.Name),
		URL:                in.URL,
		Username:           in.Username,
		PasswordEncrypted:  enc,
		InsecureSkipVerify: in.InsecureSkipVerify,
	}
	if err := s.regs.Create(reg); err != nil {
		return nil, err
	}
	rid := reg.ID
	aid := actorID
	_ = s.audit.Create(&models.AuditLog{
		UserID: &aid, Username: actorName, Action: models.ActionCreateRegistry,
		RegistryID: &rid, Metadata: fmt.Sprintf(`{"name":"%s","url":"%s"}`, reg.Name, reg.URL), Result: "success",
	})
	utils.LogAction(actorName, models.ActionCreateRegistry, reg.Name, "", "", "success", nil)
	resp := reg.ToResponse()
	return &resp, nil
}

func (s *RegistryService) Update(actorID uint, actorName string, id uint, in UpdateRegistryInput) (*models.RegistryResponse, error) {
	reg, err := s.regs.FindByID(id)
	if err != nil {
		return nil, err
	}
	if in.Name != nil {
		reg.Name = strings.TrimSpace(*in.Name)
	}
	if in.URL != nil {
		u := strings.TrimRight(strings.TrimSpace(*in.URL), "/")
		if err := validateRegistryURL(u); err != nil {
			return nil, err
		}
		reg.URL = u
	}
	if in.Username != nil {
		reg.Username = *in.Username
	}
	if in.Password != nil {
		enc, err := utils.Encrypt(*in.Password, s.cfg.EncryptionKey)
		if err != nil {
			return nil, err
		}
		reg.PasswordEncrypted = enc
	}
	if in.InsecureSkipVerify != nil {
		reg.InsecureSkipVerify = *in.InsecureSkipVerify
	}
	if err := s.regs.Update(reg); err != nil {
		return nil, err
	}
	rid := reg.ID
	aid := actorID
	_ = s.audit.Create(&models.AuditLog{
		UserID: &aid, Username: actorName, Action: models.ActionUpdateRegistry,
		RegistryID: &rid, Result: "success",
	})
	resp := reg.ToResponse()
	return &resp, nil
}

func (s *RegistryService) Delete(actorID uint, actorName string, id uint) error {
	reg, err := s.regs.FindByID(id)
	if err != nil {
		return err
	}
	if err := s.regs.Delete(id); err != nil {
		return err
	}
	rid := reg.ID
	aid := actorID
	_ = s.audit.Create(&models.AuditLog{
		UserID: &aid, Username: actorName, Action: models.ActionDeleteRegistry,
		RegistryID: &rid, Metadata: fmt.Sprintf(`{"name":"%s"}`, reg.Name), Result: "success",
	})
	utils.LogAction(actorName, models.ActionDeleteRegistry, reg.Name, "", "", "success", nil)
	return nil
}

func (s *RegistryService) Test(actorID uint, actorName string, id uint) error {
	reg, err := s.regs.FindByID(id)
	if err != nil {
		return err
	}
	client, err := s.clientFor(reg)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	err = client.Ping(ctx)
	result := "success"
	if err != nil {
		result = "failure"
	}
	rid := reg.ID
	aid := actorID
	_ = s.audit.Create(&models.AuditLog{
		UserID: &aid, Username: actorName, Action: models.ActionTestRegistry,
		RegistryID: &rid, Result: result,
	})
	utils.LogAction(actorName, models.ActionTestRegistry, reg.Name, "", "", result, err)
	return err
}

func (s *RegistryService) Health(id uint) (*registry.HealthResult, error) {
	reg, err := s.regs.FindByID(id)
	if err != nil {
		return nil, err
	}
	client, err := s.clientFor(reg)
	if err != nil {
		return &registry.HealthResult{Status: registry.HealthUnhealthy, Message: "failed to create client"}, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	start := time.Now()
	err = client.Ping(ctx)
	latency := time.Since(start).Milliseconds()
	if err == nil {
		return &registry.HealthResult{Status: registry.HealthHealthy, LatencyMs: latency}, nil
	}
	if errors.Is(err, registry.ErrUnauthorized) {
		return &registry.HealthResult{Status: registry.HealthUnauthorized, LatencyMs: latency, Message: err.Error()}, nil
	}
	if errors.Is(err, registry.ErrUnreachable) {
		return &registry.HealthResult{Status: registry.HealthUnreachable, LatencyMs: latency, Message: err.Error()}, nil
	}
	return &registry.HealthResult{Status: registry.HealthUnhealthy, LatencyMs: latency, Message: err.Error()}, nil
}

type RepoSummary struct {
	Name      string `json:"name"`
	TagCount  int    `json:"tag_count"`
	LatestTag string `json:"latest_tag,omitempty"`
}

type TagInfo struct {
	Tag     string `json:"tag"`
	Digest  string `json:"digest,omitempty"`
	Created string `json:"created,omitempty"`
	Size    int64  `json:"size,omitempty"`
}

func (s *RegistryService) ListRepositories(id uint, search string) ([]RepoSummary, error) {
	reg, err := s.regs.FindByID(id)
	if err != nil {
		return nil, err
	}
	client, err := s.clientFor(reg)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	repos, err := client.Catalog(ctx)
	if err != nil {
		return nil, mapRegistryErr(err)
	}
	search = strings.ToLower(strings.TrimSpace(search))
	out := make([]RepoSummary, 0, len(repos))
	for _, name := range repos {
		if search != "" && !strings.Contains(strings.ToLower(name), search) {
			continue
		}
		tags, err := client.ListTags(ctx, name)
		if err != nil {
			out = append(out, RepoSummary{Name: name, TagCount: 0})
			continue
		}
		latest := ""
		if len(tags) > 0 {
			sort.Strings(tags)
			latest = tags[len(tags)-1]
		}
		out = append(out, RepoSummary{Name: name, TagCount: len(tags), LatestTag: latest})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (s *RegistryService) ListTags(id uint, repository string) ([]TagInfo, error) {
	reg, err := s.regs.FindByID(id)
	if err != nil {
		return nil, err
	}
	client, err := s.clientFor(reg)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	tags, err := client.ListTags(ctx, repository)
	if err != nil {
		return nil, mapRegistryErr(err)
	}
	out := make([]TagInfo, 0, len(tags))
	for _, tag := range tags {
		info := TagInfo{Tag: tag}
		m, err := client.GetManifest(ctx, repository, tag)
		if err == nil && m != nil {
			info.Digest = m.Digest
			info.Created = m.Created
			info.Size = m.TotalSize
		}
		out = append(out, info)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Tag > out[j].Tag })
	return out, nil
}

func (s *RegistryService) GetManifest(id uint, repository, reference string) (*registry.Manifest, error) {
	reg, err := s.regs.FindByID(id)
	if err != nil {
		return nil, err
	}
	client, err := s.clientFor(reg)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	m, err := client.GetManifest(ctx, repository, reference)
	if err != nil {
		return nil, mapRegistryErr(err)
	}
	return m, nil
}

func (s *RegistryService) DeleteImage(actorID uint, actorName string, id uint, repository, tag string) error {
	reg, err := s.regs.FindByID(id)
	if err != nil {
		return err
	}
	client, err := s.clientFor(reg)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	m, err := client.GetManifest(ctx, repository, tag)
	if err != nil {
		return mapRegistryErr(err)
	}
	if m.Digest == "" {
		return errors.New("could not resolve manifest digest for tag")
	}
	if err := client.DeleteManifest(ctx, repository, m.Digest); err != nil {
		utils.LogAction(actorName, models.ActionDeleteImage, reg.Name, repository, m.Digest, "failure", err)
		return mapRegistryErr(err)
	}
	rid := reg.ID
	aid := actorID
	_ = s.audit.Create(&models.AuditLog{
		UserID: &aid, Username: actorName, Action: models.ActionDeleteImage,
		RegistryID: &rid, Repository: repository, Tag: tag, Digest: m.Digest, Result: "success",
	})
	utils.LogAction(actorName, models.ActionDeleteImage, reg.Name, repository, m.Digest, "success", nil)
	return nil
}

func (s *RegistryService) RunGC(actorID uint, actorName string, id uint, dryRun bool) (*registry.GCResult, error) {
	reg, err := s.regs.FindByID(id)
	if err != nil {
		return nil, err
	}
	mode := registry.GCMode(s.cfg.RegistryGCMode)
	var result *registry.GCResult
	if mode == registry.GCModeRemote {
		result = registry.RemoteGCUnsupported()
	} else {
		runner := &registry.LocalGCRunner{ContainerName: s.cfg.RegistryContainer}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		result, err = runner.Run(ctx, dryRun)
		if err != nil && result == nil {
			return nil, err
		}
	}
	rid := reg.ID
	aid := actorID
	res := "success"
	if result != nil && !result.Supported {
		res = "unsupported"
	} else if err != nil {
		res = "failure"
	}
	_ = s.audit.Create(&models.AuditLog{
		UserID: &aid, Username: actorName, Action: models.ActionRunGC,
		RegistryID: &rid,
		Metadata:   fmt.Sprintf(`{"dry_run":%v,"supported":%v}`, dryRun, result != nil && result.Supported),
		Result:     res,
	})
	utils.LogAction(actorName, models.ActionRunGC, reg.Name, "", "", res, err)
	return result, nil
}

type CleanupPreview struct {
	Repository             string   `json:"repository"`
	TagsWouldDelete        []string `json:"tags_would_delete"`
	Count                  int      `json:"count"`
	EstimatedReclaimBytes  int64    `json:"estimated_reclaim_bytes"`
	EstimatedReclaimHuman  string   `json:"estimated_reclaim_human"`
	KeepLatest             int      `json:"keep_latest"`
	DeleteOlderDays        int      `json:"delete_older_days"`
}

func (s *RegistryService) PreviewCleanup(actorID uint, actorName string, id uint, repository string, keepLatest, deleteOlderDays int) (*CleanupPreview, error) {
	if keepLatest < 0 {
		keepLatest = 0
	}
	reg, err := s.regs.FindByID(id)
	if err != nil {
		return nil, err
	}
	client, err := s.clientFor(reg)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	tags, err := client.ListTags(ctx, repository)
	if err != nil {
		return nil, mapRegistryErr(err)
	}

	type tagged struct {
		tag     string
		created time.Time
		size    int64
	}
	items := make([]tagged, 0, len(tags))
	for _, tag := range tags {
		m, err := client.GetManifest(ctx, repository, tag)
		t := tagged{tag: tag}
		if err == nil && m != nil {
			t.size = m.TotalSize
			if m.Created != "" {
				if parsed, e := time.Parse(time.RFC3339Nano, m.Created); e == nil {
					t.created = parsed
				} else if parsed, e := time.Parse(time.RFC3339, m.Created); e == nil {
					t.created = parsed
				}
			}
		}
		items = append(items, t)
	}

	sort.Slice(items, func(i, j int) bool {
		if !items[i].created.IsZero() && !items[j].created.IsZero() {
			return items[i].created.After(items[j].created)
		}
		return items[i].tag > items[j].tag
	})

	keep := map[string]bool{}
	for i := 0; i < len(items) && i < keepLatest; i++ {
		keep[items[i].tag] = true
	}

	cutoff := time.Time{}
	if deleteOlderDays > 0 {
		cutoff = time.Now().AddDate(0, 0, -deleteOlderDays)
	}

	var wouldDelete []string
	var reclaim int64
	for _, it := range items {
		if keep[it.tag] {
			continue
		}
		if deleteOlderDays > 0 {
			if it.created.IsZero() || !it.created.Before(cutoff) {
				continue
			}
		} else if keepLatest > 0 {
			// keep latest only mode: delete anything not in keep set
		} else {
			continue
		}
		wouldDelete = append(wouldDelete, it.tag)
		reclaim += it.size
	}

	preview := &CleanupPreview{
		Repository:            repository,
		TagsWouldDelete:       wouldDelete,
		Count:                 len(wouldDelete),
		EstimatedReclaimBytes: reclaim,
		EstimatedReclaimHuman: formatBytes(reclaim),
		KeepLatest:            keepLatest,
		DeleteOlderDays:       deleteOlderDays,
	}

	rid := reg.ID
	aid := actorID
	_ = s.audit.Create(&models.AuditLog{
		UserID: &aid, Username: actorName, Action: models.ActionPreviewCleanup,
		RegistryID: &rid, Repository: repository,
		Metadata: fmt.Sprintf(`{"count":%d}`, preview.Count), Result: "success",
	})

	_ = s.retention.Upsert(&models.RetentionPolicy{
		RegistryID:      id,
		Repository:      repository,
		KeepLatest:      keepLatest,
		DeleteOlderDays: deleteOlderDays,
	})

	return preview, nil
}

type StorageInfo struct {
	Available bool   `json:"available"`
	Message   string `json:"message"`
	Detail    string `json:"detail"`
}

func defaultStorageInfo() StorageInfo {
	return StorageInfo{
		Available: false,
		Message:   "Storage information unavailable",
		Detail:    "Docker Registry HTTP API V2 does not expose filesystem or disk usage. RegPilot does not report disk metrics without a reliable registry API.",
	}
}

type DashboardStats struct {
	RegistryCount     int64                    `json:"registry_count"`
	RepositoryCount   int                      `json:"repository_count"`
	TagCount          int                      `json:"tag_count"`
	TagCountPartial   bool                     `json:"tag_count_partial"`
	Storage           StorageInfo              `json:"storage"`
	StorageAvailable  bool                     `json:"storage_available"`
	StorageMessage    string                   `json:"storage_message"`
	Registries        []DashboardRegistryStat  `json:"registries"`
	RecentActivity    []models.AuditLog        `json:"recent_activity"`
}

func (s *RegistryService) StorageInfo() StorageInfo {
	return defaultStorageInfo()
}

type DashboardRegistryStat struct {
	ID           uint                   `json:"id"`
	Name         string                 `json:"name"`
	Status       registry.HealthStatus  `json:"status"`
	LatencyMs    int64                  `json:"latency_ms"`
	Repositories int                    `json:"repositories"`
	Tags         int                    `json:"tags"`
}

const dashboardMaxReposForTagCount = 80

func (s *RegistryService) Dashboard() (*DashboardStats, error) {
	regs, err := s.regs.List()
	if err != nil {
		return nil, err
	}
	storage := defaultStorageInfo()
	stats := &DashboardStats{
		RegistryCount:    int64(len(regs)),
		Storage:          storage,
		StorageAvailable: storage.Available,
		StorageMessage:   storage.Message,
		Registries:       make([]DashboardRegistryStat, 0, len(regs)),
	}
	recent, _ := s.audit.Recent(15)
	stats.RecentActivity = recent

	if len(regs) == 0 {
		return stats, nil
	}

	type dashRow struct {
		index int
		stat  DashboardRegistryStat
		repos int
		tags  int
		partial bool
	}

	rows := make([]dashRow, len(regs))
	var wg sync.WaitGroup
	for i := range regs {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			reg := &regs[idx]
			st := DashboardRegistryStat{ID: reg.ID, Name: reg.Name, Status: registry.HealthUnreachable}
			reposN, tagsN, partial := 0, 0, false

			h, err := s.Health(reg.ID)
			if err == nil && h != nil {
				st.Status = h.Status
				st.LatencyMs = h.LatencyMs
			}

			client, err := s.clientFor(reg)
			if err == nil {
				ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
				repos, err := client.Catalog(ctx)
				if err == nil {
					reposN = len(repos)
					st.Repositories = reposN
					limit := len(repos)
					if limit > dashboardMaxReposForTagCount {
						limit = dashboardMaxReposForTagCount
						partial = true
					}
					for _, repo := range repos[:limit] {
						tags, err := client.ListTags(ctx, repo)
						if err == nil {
							tagsN += len(tags)
						}
					}
					st.Tags = tagsN
				}
				cancel()
			}
			rows[idx] = dashRow{index: idx, stat: st, repos: reposN, tags: tagsN, partial: partial}
		}(i)
	}
	wg.Wait()

	for _, row := range rows {
		stats.RepositoryCount += row.repos
		stats.TagCount += row.tags
		if row.partial {
			stats.TagCountPartial = true
		}
		stats.Registries = append(stats.Registries, row.stat)
	}
	sort.Slice(stats.Registries, func(i, j int) bool {
		return stats.Registries[i].ID < stats.Registries[j].ID
	})
	return stats, nil
}

func mapRegistryErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, registry.ErrUnauthorized) {
		return err
	}
	if errors.Is(err, registry.ErrUnreachable) {
		return err
	}
	if errors.Is(err, registry.ErrNotFound) {
		return err
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return err
}

func formatBytes(b int64) string {
	if b <= 0 {
		return "0 B"
	}
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
