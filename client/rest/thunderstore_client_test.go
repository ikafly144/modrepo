package rest

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ikafly144/modrepo/common/rest/model"
	"github.com/ikafly144/modrepo/pkg/thunderstore"
)

func TestPackageVersionToModVersion_NormalizesDependencyConstraint(t *testing.T) {
	pkgVer := &thunderstore.PackageVersion{
		VersionNumber: "0.5.0",
		FullName:      "giosuel-Imperium_Repo-0.5.0",
		Dependencies: []string{
			"BepInEx-BepInExPack-5.4.2305",
			"DiFFoZ-BepInEx_MonoMod_Debug_Patcher-1.1.1",
		},
	}

	mv := packageVersionToModVersion(pkgVer, "giosuel-Imperium_Repo")
	require.Len(t, mv.Dependencies, 2)

	require.Equal(t, "BepInEx-BepInExPack", mv.Dependencies[0].ModID)
	require.Equal(t, ">=5.4.2305", mv.Dependencies[0].VersionID)
	require.Equal(t, model.DependencyTypeRequired, mv.Dependencies[0].DependencyType)

	require.Equal(t, "DiFFoZ-BepInEx_MonoMod_Debug_Patcher", mv.Dependencies[1].ModID)
	require.Equal(t, ">=1.1.1", mv.Dependencies[1].VersionID)
}
