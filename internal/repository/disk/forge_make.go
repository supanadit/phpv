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

	if pkgName == "automake" || pkgName == "autoconf" || pkgName == "libtool" {
		jobs = 1
	} else {
		jobs = utils.GetJobs(jobs)
	}

	args := []string{fmt.Sprintf("-j%d", jobs)}
	if pkgName == "automake" || pkgName == "autoconf" || pkgName == "libtool" {
		args = append(args, "MAKEINFO=true", "HELP2MAN=true")
	}

	mk := ctx.Command("make", args...)
	mk.Dir = sourcePath
	mk.Env = env

	return ctx.Run(mk)
}

func (r *ForgeRepository) makeInstall(sourcePath string, jobs int, env []string, verbose bool, pkgName string) error {
	ctx := utils.NewExecContext(verbose)
	jobs = utils.GetJobs(jobs)

	installTarget := "install"
	if pkgName == "openssl" || pkgName == "ossl" {
		installTarget = "install_sw"
	}

	args := []string{fmt.Sprintf("-j%d", jobs), installTarget}
	if pkgName == "automake" || pkgName == "autoconf" || pkgName == "libtool" {
		args = append(args, "MAKEINFO=true", "HELP2MAN=true")
	}

	mkInstall := ctx.Command("make", args...)
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
