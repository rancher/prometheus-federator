package debug

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
)

func ChartDebugSubCommand(base64ChartTarball string) *cobra.Command {
	var chartOnly bool
	chartDebug := &cobra.Command{
		Use:   "debug-chart",
		Short: "This command helps debug the internal chart files",
		RunE: func(_ *cobra.Command, _ []string) error {
			if chartOnly {
				logrus.Info("Only the embedded chart's `Chart.yaml` will be exported.")
			} else {
				logrus.Info("The entire embedded chart wil be exported.")
			}

			tarReader, err := readEmbeddedHelmChart(base64ChartTarball)
			if err != nil {
				return err
			}

			// Extract files to the current working directory
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("error getting current working directory: %v", err)
			}
			debugDir := filepath.Join(cwd, ".debug")
			// Ensure the .debug directory exists
			if err := os.MkdirAll(debugDir, 0755); err != nil {
				return fmt.Errorf("error creating .debug directory: %v", err)
			}

			err = extractChartData(tarReader, debugDir, chartOnly)
			if err != nil {
				return err
			}

			logrus.Infof("Chart files successfully extracted to: %s", debugDir)
			return nil
		},
	}
	chartDebug.Flags().BoolVarP(&chartOnly, "chart-only", "C", false, "When set, only the `Chart.yaml` will be exported.")
	return chartDebug
}

func readEmbeddedHelmChart(base64ChartTarball string) (*tar.Reader, error) {
	chartTarballData, err := base64.StdEncoding.DecodeString(base64ChartTarball)
	if err != nil {
		return nil, fmt.Errorf("error reading embedded chart data from base64: %v", err)
	}

	gzipReader, err := gzip.NewReader(bytes.NewReader(chartTarballData))
	if err != nil {
		return nil, fmt.Errorf("error creating gzip reader: %v", err)
	}
	defer gzipReader.Close()

	// Create a tar reader
	tarReader := tar.NewReader(gzipReader)
	return tarReader, nil
}

func extractChartData(tarReader *tar.Reader, debugDir string, chartOnly bool) error {
	// Extract files to the .debug directory
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			// End of archive
			break
		}
		if err != nil {
			return fmt.Errorf("error reading tarball: %v", err)
		}

		// Determine the path to extract the file to
		filePath := filepath.Join(debugDir, header.Name)

		if chartOnly {
			err = extractChartYamlFile(tarReader, header, filePath)
		} else {
			err = extractAllChartData(tarReader, header, filePath)
		}

		if err != nil {
			return err
		}
	}
	return nil
}

func extractChartYamlFile(tarReader *tar.Reader, header *tar.Header, filePath string) error {
	switch header.Typeflag {
	case tar.TypeReg:
		if header.Name == "rancher-project-monitoring/Chart.yaml" {
			logrus.Info("Found a `Chart.yaml` file to export.")
			// Ensure the parent directory exists
			parentDir := filepath.Dir(filePath)
			if err := os.MkdirAll(parentDir, 0755); err != nil {
				return fmt.Errorf("error creating parent directory: %v", err)
			}

			// Create regular file
			outFile, err := os.Create(filePath)
			if err != nil {
				return fmt.Errorf("error creating file: %v", err)
			}

			// Copy file content
			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return fmt.Errorf("error writing file content: %v", err)
			}
			logrus.Info("The `Chart.yaml` file was exported")
			outFile.Close()
		} else {
			logrus.Debugf("Skipping file: %s\n", header.Name)
		}
	default:
		logrus.Debugf("Skipping file: %s\n", header.Name)
	}
	return nil
}

func extractAllChartData(tarReader *tar.Reader, header *tar.Header, filePath string) error {
	switch header.Typeflag {
	case tar.TypeDir:
		// Create directory
		if err := os.MkdirAll(filePath, os.FileMode(header.Mode)); err != nil {
			return fmt.Errorf("error creating directory: %v", err)
		}
	case tar.TypeReg:
		// Ensure the parent directory exists
		parentDir := filepath.Dir(filePath)
		if err := os.MkdirAll(parentDir, 0755); err != nil {
			return fmt.Errorf("error creating parent directory: %v", err)
		}

		// Create regular file
		outFile, err := os.Create(filePath)
		if err != nil {
			return fmt.Errorf("error creating file: %v", err)
		}

		// Copy file content
		if _, err := io.Copy(outFile, tarReader); err != nil {
			outFile.Close()
			return fmt.Errorf("error writing file content: %v", err)
		}
		outFile.Close()
	default:
		logrus.Debugf("Skipping unsupported file type: %s\n", header.Name)
	}
	return nil
}
