package core

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	restcommon "github.com/ikafly144/modrepo/common/rest"
	"github.com/ikafly144/modrepo/common/rest/model"
	"github.com/ikafly144/modrepo/pkg/modmgr"
	"github.com/ikafly144/modrepo/pkg/thunderstore"
)

type mockRestVersionProvider struct {
	versions map[string]map[string]modmgr.ModVersion
	ids      map[string][]string
	latest   map[string]string
}

func (m *mockRestVersionProvider) ServerBaseURL() string { return "" }
func (m *mockRestVersionProvider) GetHealthStatus() (*restcommon.HealthStatus, error) {
	return nil, nil
}

func (m *mockRestVersionProvider) GetVersionInfo() (*restcommon.VersionInfo, error) {
	return nil, nil
}

func (m *mockRestVersionProvider) GetModVersion(modID string, versionID string) (*modmgr.ModVersion, error) {
	if vers, ok := m.versions[modID]; ok {
		if v, ok := vers[versionID]; ok {
			return &v, nil
		}
	}
	return nil, fmt.Errorf("version not found: %s@%s", modID, versionID)
}

func (m *mockRestVersionProvider) GetLatestModVersion(modID string) (*modmgr.ModVersion, error) {
	if lat, ok := m.latest[modID]; ok {
		return m.GetModVersion(modID, lat)
	}
	return nil, fmt.Errorf("latest not found: %s", modID)
}

func (m *mockRestVersionProvider) GetModVersionIDs(modID string, limit int, after string) ([]string, error) {
	if ids, ok := m.ids[modID]; ok {
		return ids, nil
	}
	return nil, fmt.Errorf("mod not found: %s", modID)
}

func (m *mockRestVersionProvider) GetModIDs(limit int, after string, before string) ([]string, error) {
	return nil, nil
}
func (m *mockRestVersionProvider) GetMod(modID string) (*modmgr.Mod, error) {
	return nil, nil
}
func (m *mockRestVersionProvider) GetModThumbnail(modID string) ([]byte, error) {
	return nil, nil
}
func (m *mockRestVersionProvider) CheckForUpdates(installed map[string]string) (map[string]*modmgr.ModVersion, error) {
	return nil, nil
}
func (m *mockRestVersionProvider) SearchMods(query, category, sortBy string) ([]*modmgr.Mod, error) {
	return nil, nil
}
func (m *mockRestVersionProvider) GetCategories() []string { return nil }
func (m *mockRestVersionProvider) RefreshPackages(ctx context.Context) error {
	return nil
}
func (m *mockRestVersionProvider) ThunderstoreClient() *thunderstore.Client {
	return nil
}

func TestApp_ResolveDependencies_NormalizesProfileConstraints(t *testing.T) {
	app := &App{}

	mockRest := &mockRestVersionProvider{
		versions: map[string]map[string]modmgr.ModVersion{
			"BepInEx-BepInExPack": {
				"5.4.2100": {ModID: "BepInEx-BepInExPack", VersionID: "5.4.2100"},
				"5.4.2305": {ModID: "BepInEx-BepInExPack", VersionID: "5.4.2305"},
			},
			"dep-mod-a": {
				"1.0.0": {
					ModID:     "dep-mod-a",
					VersionID: "1.0.0",
					Dependencies: []model.ModVersionDependency{
						{ModID: "BepInEx-BepInExPack", VersionID: ">=5.4.2100", DependencyType: model.DependencyTypeRequired},
					},
				},
			},
		},
		ids: map[string][]string{
			"BepInEx-BepInExPack": {"5.4.2305", "5.4.2100"},
			"dep-mod-a":           {"1.0.0"},
		},
		latest: map[string]string{
			"BepInEx-BepInExPack": "5.4.2305",
			"dep-mod-a":           "1.0.0",
		},
	}
	app.Rest = mockRest

	// Simulate profile containing Imperium_Repo which has bare constraint "5.4.2305" and depends on dep-mod-a
	initialMods := []modmgr.ModVersion{
		{
			ModID:     "giosuel-Imperium_Repo",
			VersionID: "0.5.0",
			Dependencies: []model.ModVersionDependency{
				{ModID: "BepInEx-BepInExPack", VersionID: "5.4.2305", DependencyType: model.DependencyTypeRequired},
				{ModID: "dep-mod-a", VersionID: "1.0.0", DependencyType: model.DependencyTypeRequired},
			},
		},
	}

	resolved, err := app.ResolveDependencies(initialMods)
	require.NoError(t, err)

	resolvedMap := make(map[string]string)
	for _, v := range resolved {
		resolvedMap[v.ModID] = v.VersionID
	}

	require.Equal(t, "5.4.2305", resolvedMap["BepInEx-BepInExPack"])
	require.Equal(t, "1.0.0", resolvedMap["dep-mod-a"])
	require.Equal(t, "0.5.0", resolvedMap["giosuel-Imperium_Repo"])
}
