package yarnstart

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"bufio"

	"github.com/paketo-buildpacks/libnodejs"
	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/fs"
)

// NoStartScriptError indicates that the targeted project does no have a start command in their package.json
const NoStartScriptError = "no start script in package.json"

func Detect() packit.DetectFunc {
	return func(context packit.DetectContext) (packit.DetectResult, error) {
		projectPath, err := libnodejs.FindProjectPath(context.WorkingDir)
		if err != nil {
			return packit.DetectResult{}, err
		}

		exists, err := fs.Exists(filepath.Join(projectPath, "yarn.lock"))
		if err != nil {
			return packit.DetectResult{}, fmt.Errorf("failed to stat yarn.lock: %w", err)
		}

		if !exists {
			return packit.DetectResult{}, packit.Fail.WithMessage("no 'yarn.lock' found in the project path %s", projectPath)
		}

		pkg, err := libnodejs.ParsePackageJSON(projectPath)
		if err != nil {
			if os.IsNotExist(err) {
				return packit.DetectResult{}, packit.Fail.WithMessage("no 'package.json' found in project path %s", projectPath)
			}
			return packit.DetectResult{}, fmt.Errorf("failed to open package.json: %w", err)
		}

		if pkg.HasStartScript() {
			fmt.Println("===============> Start script:", pkg.Scripts.Start)
		}

		launchNodeModules := !checkSlugIgnore()

		requirements := []packit.BuildPlanRequirement{
			{
				Name: Node,
				Metadata: map[string]interface{}{
					"launch": true,
				},
			},
			{
				Name: Yarn,
				Metadata: map[string]interface{}{
					"launch": true,
				},
			},
			{
				Name: NodeModules,
				Metadata: map[string]interface{}{
					"launch": launchNodeModules,
				},
			},
		}

		shouldReload, err := checkLiveReloadEnabled()
		if err != nil {
			return packit.DetectResult{}, err
		}

		if shouldReload {
			requirements = append(requirements, packit.BuildPlanRequirement{
				Name: "watchexec",
				Metadata: map[string]interface{}{
					"launch": true,
				},
			})
		}

		return packit.DetectResult{
			Plan: packit.BuildPlan{
				Requires: requirements,
			},
		}, nil
	}
}

func checkLiveReloadEnabled() (bool, error) {
	if reload, ok := os.LookupEnv("BP_LIVE_RELOAD_ENABLED"); ok {
		shouldEnableReload, err := strconv.ParseBool(reload)
		if err != nil {
			return false, fmt.Errorf("failed to parse BP_LIVE_RELOAD_ENABLED value %s: %w", reload, err)
		}
		return shouldEnableReload, nil
	}
	return false, nil
}


func checkSlugIgnore() bool {
	filename := ".slugignore"

	// Check if the file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return false
	}

	// Open the file
	file, err := os.Open(filename)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return false
	}
	defer file.Close()

	// Check if "/node_modules" exists as an exact line
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if scanner.Text() == "/node_modules" {
			return true
		}
	}

	// Handle potential scanner errors
	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading file:", err)
	}

	return false
}

