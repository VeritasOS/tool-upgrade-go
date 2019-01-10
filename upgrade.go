package upgrade

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/blang/semver"
	"github.com/pkg/errors"
	"github.com/termie/go-shutil"
)

func CurrentVersion(version string) (semver.Version, error) {
	return semver.Make(version)
}

func AvailableVersion(repo string, filePrefix string, channel string) (semver.Version, error) {
	resp, err := http.Get(
		fmt.Sprintf("%s/%s%s", repo, filePrefix, channel),
	)
	if err != nil {
		return semver.Version{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return semver.Version{}, fmt.Errorf(
			"while getting version; unexpected status %s",
			resp.Status,
		)
	}
	versionBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return semver.Version{}, err
	}

	return semver.Make(string(versionBytes))
}

// CheckAndNotifyIfOutOfDate checks if this utility is out of date compared to
// the repository. It uses a caching strategy to avoid costly
// checks with every run.
func CheckAndNotifyIfOutOfDate(tool string, currentVersion string, repo string, filePrefix string, versionStable string, hoursToCheckForUpdate float64, upgradeCommand string) (bool, error) {
	fn := filepath.Join(GetHome(), "."+tool+"-version-check")
	file, err := os.OpenFile(fn, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: cannot create file (%s) to cache version checks\n", fn)
		fmt.Fprintf(os.Stderr, "error: %s\n", err.Error())
		return false, err
	}
	defer file.Close()

	fi, err := file.Stat()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: cannot stat (%s)\n", fn)
		fmt.Fprintf(os.Stderr, "error: %s\n", err.Error())
		return false, err
	}

	diff := time.Now().Sub(fi.ModTime())
	if diff.Hours() < hoursToCheckForUpdate {
		return false, nil
	}

	// update mtime
	err = file.Truncate(0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: cannot update mtime (%s)\n", fn)
		fmt.Fprintf(os.Stderr, "error: %s\n", err.Error())
		return false, err
	}

	curr, err := CurrentVersion(currentVersion)
	if err != nil {
		return false, err
	}
	avail, err := AvailableVersion(repo, filePrefix, versionStable)
	if err != nil {
		return false, err
	}

	if avail.GT(curr) {
		fmt.Fprintf(os.Stderr, "A new version of [%s] is available, run: %s\n", tool, upgradeCommand)
		return true, nil
	}

	return false, nil
}

func Download(repo string, version string, tool string) (string, error) {
	// now that we have the version, create the temp file and set perms
	tmp, err := ioutil.TempFile("", tool+"_upgrade")
	if err != nil {
		return "", err
	}

	if err = tmp.Chmod(0755); err != nil {
		if err.(*os.PathError).Err.Error() != "not supported by windows" {
			return "", err
		}
	}

	tmpfn := tmp.Name()
	defer func() {
		tmp.Close()
		if err != nil {
			os.Remove(tmpfn)
		}
	}()

	// get the fresh bits
	resp, err := http.Get(
		fmt.Sprintf("%s/%s/%s_%s_%s",
			repo,
			version,
			tool,
			runtime.GOOS,
			runtime.GOARCH,
		),
	)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf(
			"while getting updated binary; unexpected status %s",
			resp.Status,
		)
	}

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		return "", err
	}

	return tmpfn, nil
}

func RemoveBackup() (string, string, error) {
	arg0, err := os.Executable()
	if err != nil {
		return "", "", errors.Wrap(err, "unable to find executable")
	}

	backup := arg0 + "~"

	err = os.RemoveAll(backup)
	if err != nil {
		return arg0, backup, errors.Wrap(err, fmt.Sprintf(
			"unable to remove %s",
			backup,
		))
	}

	return arg0, backup, nil
}

func Upgrade(tool string, currentVersion string, repo string, filePrefix string, versionStable string, upgradeForce *bool) error {
	curr, err := CurrentVersion(currentVersion)
	if err != nil {
		return errors.Wrap(err, "unable to get current version")
	}
	avail, err := AvailableVersion(repo, filePrefix, versionStable)
	if err != nil {
		return errors.Wrap(err, "unable to get available version")
	}

	if !*upgradeForce && curr.GTE(avail) {
		fmt.Printf("%s is up-to-date. Go forth and be awesome!\n", tool)
		return nil
	}

	arg0, backup, err := RemoveBackup()
	if err != nil {
		return err
	}

	fmt.Println("upgrading", arg0, "to", avail)
	tmp, err := Download(repo, avail.String(), tool)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf(
			"unable to upgrade %s",
			tool,
		))
	}

	err = os.Rename(arg0, backup)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf(
			"unable to upgrade %s in-place; move %s to %s",
			tool,
			tmp,
			os.Args[0],
		))
	}

	_, err = shutil.Copy(tmp, arg0, false)
	if err != nil {
		err = os.Rename(backup, arg0)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf(
				"upgrade failed and unable to recover from backup;"+
					" move %s to %s",
				tmp,
				os.Args[0],
			))
		}
		return errors.Wrap(err, fmt.Sprintf(
			"unable to upgrade %s in-place; move %s to %s",
			tool,
			tmp,
			os.Args[0],
		))
	}

	err = os.Remove(tmp)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf(
			"unable to clean up temp file; remove %s",
			tmp,
		))
	}

	return nil
}
