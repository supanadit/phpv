package memory

import (
	"context"
	"sort"

	"github.com/supanadit/phpv/domain"
)

type VersionRepository struct {
}

func NewVersionRepository() *VersionRepository {
	return &VersionRepository{}
}

func (r *VersionRepository) GetVersions(ctx context.Context) ([]domain.Version, error) {
	// TODO: Add RCx, beta, alpha versions
	versions := r.generateRangeVersions(8, 4, 0, 14)
	versions = append(
		versions,
		r.generateRangeVersions(8, 3, 0, 27)...,
	)
	versions = append(
		versions,
		r.generateRangeVersions(8, 2, 0, 29)...,
	)
	versions = append(
		versions,
		r.generateRangeVersions(8, 1, 0, 33)...,
	)
	versions = append(
		versions,
		r.generateRangeVersions(8, 0, 0, 30)...,
	)
	versions = append(
		versions,
		r.generateRangeVersions(7, 4, 0, 33)...,
	)
	versions = append(
		versions,
		r.generateRangeVersions(7, 3, 0, 33)...,
	)
	versions = append(
		versions,
		r.generateRangeVersions(7, 2, 0, 34)...,
	)
	versions = append(
		versions,
		r.generateRangeVersions(7, 1, 0, 33)...,
	)
	versions = append(
		versions,
		r.generateRangeVersions(5, 6, 0, 40)...,
	)
	versions = append(
		versions,
		r.generateRangeVersions(5, 5, 0, 38)...,
	)
	versions = append(
		versions,
		r.generateRangeVersions(5, 4, 0, 45)...,
	)
	versions = append(
		versions,
		r.generateRangeVersions(5, 3, 0, 29)...,
	)
	versions = append(
		versions,
		r.generateRangeVersions(5, 2, 0, 17)...,
	)
	versions = append(
		versions,
		r.generateRangeVersions(5, 1, 0, 6)...,
	)
	versions = append(
		versions,
		r.generateRangeVersions(5, 0, 0, 5)...,
	)
	versions = append(
		versions,
		r.generateRangeVersions(4, 4, 0, 9)...,
	)
	versions = append(
		versions,
		r.generateRangeVersions(4, 3, 0, 11)...,
	)
	versions = append(
		versions,
		r.generateRangeVersions(4, 2, 0, 3)...,
	)
	versions = append(
		versions,
		r.generateRangeVersions(4, 1, 0, 2)...,
	)
	versions = append(
		versions,
		r.generateRangeVersions(4, 0, 0, 6)...,
	)
	// Sort versions descending
	sort.Slice(versions, func(i, j int) bool {
		if versions[i].Major != versions[j].Major {
			return versions[i].Major > versions[j].Major
		}
		if versions[i].Minor != versions[j].Minor {
			return versions[i].Minor > versions[j].Minor
		}
		return versions[i].Patch > versions[j].Patch
	})
	return versions, nil
}

func (r *VersionRepository) generateRangeVersions(major, minor, startPatch, endPatch int) []domain.Version {
	count := endPatch - startPatch + 1
	versions := make([]domain.Version, 0, count)
	for patch := startPatch; patch <= endPatch; patch++ {
		versions = append(versions, domain.Version{
			Major: major,
			Minor: minor,
			Patch: patch,
		})
	}
	return versions
}
