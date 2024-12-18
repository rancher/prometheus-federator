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
	chartDebug := &cobra.Command{
		Use:   "debug-chart",
		Short: "This command helps debug the internal chart files",
		Run: func(cmd *cobra.Command, _ []string) {
			chartOnly, _ := cmd.Flags().GetBool("chart-only")
			if chartOnly {
				logrus.Info("Only the embedded chart's `Chart.yaml` will be exported.")
			} else {
				logrus.Info("The entire embedded chart wil be exported.")
			}

			chartTarballData, err := base64.StdEncoding.DecodeString(base64ChartTarball)
			if err != nil {
				fmt.Println("error:", err)
				return
			}

			gzipReader, err := gzip.NewReader(bytes.NewReader(chartTarballData))
			if err != nil {
				fmt.Println("Error creating gzip reader:", err)
				return
			}
			defer gzipReader.Close()

			// Create a tar reader
			tarReader := tar.NewReader(gzipReader)

			// Extract files to the current working directory
			cwd, err := os.Getwd()
			if err != nil {
				fmt.Println("Error getting current working directory:", err)
				return
			}
			debugDir := filepath.Join(cwd, ".debug")
			// Ensure the .debug directory exists
			if err := os.MkdirAll(debugDir, 0755); err != nil {
				fmt.Println("Error creating .debug directory:", err)
				return
			}

			// Extract files to the .debug directory
			for {
				header, err := tarReader.Next()
				if err == io.EOF {
					// End of archive
					break
				}
				if err != nil {
					fmt.Println("Error reading tarball:", err)
					return
				}

				// Determine the path to extract the file to
				filePath := filepath.Join(debugDir, header.Name)

				if chartOnly {
					switch header.Typeflag {
					case tar.TypeReg:
						if header.Name == "rancher-project-monitoring/Chart.yaml" {
							logrus.Info("Found a `Chart.yaml` file to export.")
							// Ensure the parent directory exists
							parentDir := filepath.Dir(filePath)
							if err := os.MkdirAll(parentDir, 0755); err != nil {
								logrus.Error("Error creating parent directory:", err)
								return
							}

							// Create regular file
							outFile, err := os.Create(filePath)
							if err != nil {
								logrus.Error("Error creating file:", err)
								return
							}

							// Copy file content
							if _, err := io.Copy(outFile, tarReader); err != nil {
								logrus.Error("Error writing file content:", err)
								outFile.Close()
								return
							}
							logrus.Info("The `Chart.yaml` file was exported")
							outFile.Close()
						} else {
							logrus.Debugf("Skipping file: %s\n", header.Name)
						}
					default:
						logrus.Debugf("Skipping file: %s\n", header.Name)
					}

					return
				}

				switch header.Typeflag {
				case tar.TypeDir:
					// Create directory
					if err := os.MkdirAll(filePath, os.FileMode(header.Mode)); err != nil {
						fmt.Println("Error creating directory:", err)
						return
					}
				case tar.TypeReg:
					// Ensure the parent directory exists
					parentDir := filepath.Dir(filePath)
					if err := os.MkdirAll(parentDir, 0755); err != nil {
						fmt.Println("Error creating parent directory:", err)
						return
					}

					// Create regular file
					outFile, err := os.Create(filePath)
					if err != nil {
						fmt.Println("Error creating file:", err)
						return
					}

					// Copy file content
					if _, err := io.Copy(outFile, tarReader); err != nil {
						fmt.Println("Error writing file content:", err)
						outFile.Close()
						return
					}
					outFile.Close()
				default:
					fmt.Printf("Skipping unsupported file type: %s\n", header.Name)
				}
			}

			fmt.Println("Chart files successfully extracted to", debugDir)
		},
	}
	chartDebug.PersistentFlags().BoolP("chart-only", "C", false, "When set, only the `Chart.yaml` will be exported.")
	return chartDebug
}
