package util

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/schollz/progressbar/v3"
	"github.com/spf13/viper"
)

func DownloadWithProgress(resp *http.Response, dest *os.File, filename string) error {
	if viper.GetBool("PHPV_QUIET") {
		_, err := io.Copy(dest, resp.Body)
		return err
	}

	bar := progressbar.NewOptions64(
		resp.ContentLength,
		progressbar.OptionSetDescription(fmt.Sprintf("Downloading %s:", filename)),
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionShowBytes(true),
		progressbar.OptionShowCount(),
		progressbar.OptionShowTotalBytes(true),
		progressbar.OptionShowElapsedTimeOnFinish(),
		progressbar.OptionSetWidth(40),
		progressbar.OptionThrottle(100*time.Millisecond),
		progressbar.OptionOnCompletion(func() {
			fmt.Fprint(os.Stderr, "\n")
		}),
	)

	_, err := io.Copy(dest, io.TeeReader(resp.Body, bar))
	return err
}
