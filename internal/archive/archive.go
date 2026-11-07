// Package archive coordinates best-effort copying behind an independent source boundary.
package archive

import (
	"context"
	"errors"
	"path/filepath"

	"github.com/jaysonhurd/icloud-dl/internal/media"
	"github.com/jaysonhurd/icloud-dl/internal/state"
)

// Asset identifies a single supported original; URLs are deliberately excluded.
type Asset struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Size      int64  `json:"size"`
	Supported bool   `json:"supported"`
}

// Source owns protocol discovery, complete enumeration, and fresh URL resolution.
type Source interface {
	List(context.Context) ([]Asset, error)
	URL(context.Context, Asset) (string, error)
}

// Config contains the per-invocation policy, never account credentials.
type Config struct {
	Destination string `json:"destination"`
	Overwrite   bool   `json:"overwrite"`
	DryRun      bool   `json:"dry_run"`
}

// Run preserves usable catalog entries after listing failure and reports actual outcomes.
// This first pass runs serially; worker scheduling and retry policy remain pending.
func Run(ctx context.Context, source Source, client media.ClientDoer, cfg Config) (state.Report, error) {
	assets, listErr := source.List(ctx)
	report := state.Report{Version: 1, ListingComplete: listErr == nil}
	result := listErr
	seen := map[string]bool{}
	for _, asset := range assets {
		if err := ctx.Err(); err != nil {
			return report, err
		}
		if asset.ID == "" {
			result = errors.Join(result, media.ErrTransfer)
			continue
		}
		if seen[asset.ID] {
			continue
		}
		seen[asset.ID] = true
		leaf := media.Name(asset.Name, asset.ID)
		item := state.Item{ID: asset.ID, Path: filepath.Join("originals", leaf), Size: asset.Size, Outcome: "unsupported"}
		if asset.Supported {
			if cfg.DryRun {
				item.Outcome = "planned"
			} else {
				raw, err := source.URL(ctx, asset)
				if err != nil {
					item.Outcome = "failed"
					result = errors.Join(result, media.ErrTransfer)
				} else {
					item.Outcome, err = media.Copy(ctx, client, raw, filepath.Join(cfg.Destination, item.Path), asset.Size, cfg.Overwrite)
					if err != nil || item.Outcome == "conflict" {
						result = errors.Join(result, media.ErrTransfer)
					}
				}
			}
		} else {
			result = errors.Join(result, media.ErrTransfer)
		}
		report.Items = append(report.Items, item)
		if !cfg.DryRun && len(report.Items)%25 == 0 {
			if state.Save(cfg.Destination, report) != nil {
				result = errors.Join(result, state.ErrReport)
			}
		}
	}
	if !cfg.DryRun && len(report.Items) > 0 {
		if state.Save(cfg.Destination, report) != nil {
			result = errors.Join(result, state.ErrReport)
		}
	}
	return report, result
}
