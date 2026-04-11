package disk

import (
	"fmt"
	"os/exec"

	"github.com/supanadit/phpv/internal/utils"
)

func (r *ForgeRepository) makeWithName(sourcePath string, jobs int, env []string, pkgName string, verbose bool) error {
	if pkgName == "m4" {
		env = append(env, "M4_MAINTAINER_MODE=no")
	}

	ctx := utils.NewExecContext(verbose)
	jobs = utils.GetJobs(jobs)

	mk := ctx.Command("make", fmt.Sprintf("-j%d", jobs))
	mk.Dir = sourcePath
	mk.Env = env

	return ctx.Run(mk)
}

func (r *ForgeRepository) makeInstall(sourcePath string, jobs int, env []string, verbose bool) error {
	ctx := utils.NewExecContext(verbose)
	jobs = utils.GetJobs(jobs)

	mkInstall := ctx.Command("make", fmt.Sprintf("-j%d", jobs), "install")
	mkInstall.Dir = sourcePath
	mkInstall.Env = env

	return ctx.Run(mkInstall)
}

func (r *ForgeRepository) runMake(jobs int, env []string, verbose bool, args ...string) error {
	ctx := utils.NewExecContext(verbose)
	jobs = utils.GetJobs(jobs)

	args = append([]string{fmt.Sprintf("-j%d", jobs)}, args...)
	mk := ctx.Command("make", args...)

	return ctx.Run(mk)
}

func (r *ForgeRepository) runCommand(name string, args []string, dir string, env []string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	if env != nil {
		cmd.Env = env
	}
	return cmd
}
