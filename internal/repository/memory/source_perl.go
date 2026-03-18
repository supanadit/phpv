package memory

import (
	"sort"

	"github.com/supanadit/phpv/domain"
)

type PerlRepository struct{}

func NewPerlRepository() *PerlRepository {
	return &PerlRepository{}
}

func (r *PerlRepository) GetVersions() ([]domain.Source, error) {
	// Source: https://www.cpan.org/src/README.html
	versions := []domain.Source{
		r.perlSource("5.42.1"),
		r.perlSource("5.40.3"),
		r.perlSource("5.38.5"),
		r.perlSource("5.36.3"),
		r.perlSource("5.34.3"),
		r.perlSource("5.32.1"),
		r.perlSource("5.30.3"),
		r.perlSource("5.28.3"),
		r.perlSource("5.26.3"),
		r.perlSource("5.24.4"),
		r.perlSource("5.22.3"),
		r.perlSource("5.20.0"),
		r.perlSource("5.18.4"),
		r.perlSource("5.16.3"),
		r.perlSource("5.14.4"),
		r.perlSource("5.12.5"),
		r.perlSource("5.10.1"),
		r.perlSource("5.8.9"),
		r.perlSource("5.6.2"),
		{Name: "perl", Version: "5.5.30", URL: "https://www.cpan.org/src/5.0/perl5.005_03.tar.gz"},
		{Name: "perl", Version: "5.4.50", URL: "https://www.cpan.org/src/5.0/perl5.004_05.tar.gz"},
	}

	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Version > versions[j].Version
	})
	return versions, nil
}

func (r *PerlRepository) perlSource(version string) domain.Source {
	ext := "tar.gz"
	if version < "5.20.0" {
		ext = "tar.bz2"
	}
	return domain.Source{
		Name:    "perl",
		Version: version,
		URL:     "https://www.cpan.org/src/5.0/perl-" + version + "." + ext,
	}
}
