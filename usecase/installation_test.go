package usecase

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/supanadit/phpv/domain"
)

// Mock repositories for testing
type MockPHPVersionRepository struct {
	mock.Mock
}

func (m *MockPHPVersionRepository) GetAvailableVersions(ctx context.Context) ([]domain.PHPVersion, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.PHPVersion), args.Error(1)
}

func (m *MockPHPVersionRepository) GetVersionByString(ctx context.Context, version string) (domain.PHPVersion, error) {
	args := m.Called(ctx, version)
	if args.Get(0) == nil {
		return domain.PHPVersion{}, args.Error(1)
	}
	return args.Get(0).(domain.PHPVersion), args.Error(1)
}

func (m *MockPHPVersionRepository) SaveVersion(ctx context.Context, version domain.PHPVersion) error {
	args := m.Called(ctx, version)
	return args.Error(0)
}

type MockInstallationRepository struct {
	mock.Mock
}

func (m *MockInstallationRepository) GetAllInstallations(ctx context.Context) ([]domain.Installation, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Installation), args.Error(1)
}

func (m *MockInstallationRepository) GetInstallationByVersion(ctx context.Context, version domain.PHPVersion) (domain.Installation, error) {
	args := m.Called(ctx, version)
	return args.Get(0).(domain.Installation), args.Error(1)
}

func (m *MockInstallationRepository) GetActiveInstallation(ctx context.Context) (domain.Installation, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return domain.Installation{}, args.Error(1)
	}
	return args.Get(0).(domain.Installation), args.Error(1)
}

func (m *MockInstallationRepository) SaveInstallation(ctx context.Context, installation domain.Installation) error {
	args := m.Called(ctx, installation)
	return args.Error(0)
}

func (m *MockInstallationRepository) SetActiveInstallation(ctx context.Context, installation domain.Installation) error {
	args := m.Called(ctx, installation)
	return args.Error(0)
}

func (m *MockInstallationRepository) DeleteInstallation(ctx context.Context, version domain.PHPVersion) error {
	args := m.Called(ctx, version)
	return args.Error(0)
}

type MockDownloader struct {
	mock.Mock
}

func (m *MockDownloader) DownloadSource(ctx context.Context, version domain.PHPVersion, destPath string) error {
	args := m.Called(ctx, version, destPath)
	return args.Error(0)
}

type MockBuilder struct {
	mock.Mock
}

func (m *MockBuilder) Build(ctx context.Context, sourcePath string, installPath string, config map[string]string) error {
	args := m.Called(ctx, sourcePath, installPath, config)
	return args.Error(0)
}

func (m *MockBuilder) GetBuildStrategy() domain.BuildStrategy {
	return domain.BuildStrategyNative
}

type MockFileSystem struct {
	mock.Mock
}

func (m *MockFileSystem) CreateDirectory(path string) error {
	args := m.Called(path)
	return args.Error(0)
}

func (m *MockFileSystem) RemoveDirectory(path string) error {
	args := m.Called(path)
	return args.Error(0)
}

func (m *MockFileSystem) FileExists(path string) bool {
	args := m.Called(path)
	return args.Bool(0)
}

func (m *MockFileSystem) DirectoryExists(path string) bool {
	args := m.Called(path)
	return args.Bool(0)
}

func TestInstallationService_InstallVersion(t *testing.T) {
	ctx := context.Background()

	// Setup mocks
	versionRepo := new(MockPHPVersionRepository)
	installRepo := new(MockInstallationRepository)
	downloader := new(MockDownloader)
	builder := new(MockBuilder)
	fs := new(MockFileSystem)

	service := NewInstallationService(versionRepo, installRepo, downloader, builder, fs, "/tmp/phpv")

	version := domain.PHPVersion{
		Version:     "8.1.0",
		Major:       8,
		Minor:       1,
		Patch:       0,
		ReleaseType: "stable",
	}

	t.Run("successful installation", func(t *testing.T) {
		// Setup expectations
		installRepo.On("GetInstallationByVersion", ctx, mock.AnythingOfType("domain.PHPVersion")).Return(domain.Installation{}, domain.ErrNotFound)
		versionRepo.On("GetVersionByString", ctx, "8.1.0").Return(domain.PHPVersion{}, domain.ErrNotFound)
		versionRepo.On("SaveVersion", ctx, mock.AnythingOfType("domain.PHPVersion")).Return(nil)
		fs.On("CreateDirectory", mock.AnythingOfType("string")).Return(nil).Twice()
		downloader.On("DownloadSource", ctx, mock.AnythingOfType("domain.PHPVersion"), mock.AnythingOfType("string")).Return(nil)
		builder.On("Build", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("map[string]string")).Return(nil)
		installRepo.On("SaveInstallation", ctx, mock.AnythingOfType("domain.Installation")).Return(nil)

		// Execute
		err := service.InstallVersion(ctx, "8.1.0")

		// Assert
		assert.NoError(t, err)
		installRepo.AssertExpectations(t)
		versionRepo.AssertExpectations(t)
		downloader.AssertExpectations(t)
		builder.AssertExpectations(t)
		fs.AssertExpectations(t)
	})

	t.Run("version already installed", func(t *testing.T) {
		// Reset mocks
		installRepo = new(MockInstallationRepository)
		versionRepo = new(MockPHPVersionRepository)

		service = NewInstallationService(versionRepo, installRepo, downloader, builder, fs, "/tmp/phpv")

		existing := domain.Installation{Version: version}
		installRepo.On("GetInstallationByVersion", ctx, mock.AnythingOfType("domain.PHPVersion")).Return(existing, nil)

		// Execute
		err := service.InstallVersion(ctx, "8.1.0")

		// Assert
		assert.Equal(t, domain.ErrConflict, err)
		installRepo.AssertExpectations(t)
	})

	t.Run("invalid version format", func(t *testing.T) {
		// Execute
		err := service.InstallVersion(ctx, "invalid")

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid version format")
	})
}

func TestInstallationService_SwitchVersion(t *testing.T) {
	ctx := context.Background()

	// Setup mocks
	installRepo := new(MockInstallationRepository)
	service := NewInstallationService(nil, installRepo, nil, nil, nil, "/tmp/phpv")

	version := domain.PHPVersion{
		Version:     "8.1.0",
		Major:       8,
		Minor:       1,
		Patch:       0,
		ReleaseType: "stable",
	}

	t.Run("successful switch", func(t *testing.T) {
		installation := domain.Installation{Version: version, Path: "/tmp/phpv/versions/8.1.0"}

		installRepo.On("GetInstallationByVersion", ctx, mock.AnythingOfType("domain.PHPVersion")).Return(installation, nil)
		installRepo.On("GetActiveInstallation", ctx).Return(domain.Installation{}, domain.ErrNotFound)
		installRepo.On("SetActiveInstallation", ctx, mock.AnythingOfType("domain.Installation")).Return(nil)

		// Execute
		err := service.SwitchVersion(ctx, "8.1.0")

		// Assert
		assert.NoError(t, err)
		installRepo.AssertExpectations(t)
	})

	t.Run("version not installed", func(t *testing.T) {
		// Create new mock for this test
		installRepo := new(MockInstallationRepository)
		service := NewInstallationService(nil, installRepo, nil, nil, nil, "/tmp/phpv")

		installRepo.On("GetInstallationByVersion", ctx, mock.AnythingOfType("domain.PHPVersion")).Return(domain.Installation{}, domain.ErrNotFound)

		// Execute
		err := service.SwitchVersion(ctx, "8.1.0")

		// Assert
		assert.Equal(t, domain.ErrNotFound, err)
		installRepo.AssertExpectations(t)
	})
}
