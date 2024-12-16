package debug

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"github.com/spf13/cobra"
	"io"
	"os"
	"path/filepath"
)

func ChartDebugSubCommand(base64ChartTarball string) *cobra.Command {
	return &cobra.Command{
		Use:   "debug-chart",
		Short: "This command helps debug the internal chart files",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("TODO debug")
			// TODO: use straing data from base64ChartTarball to:
			// Un-base64 the data to get a raw tgz file,
			// Prompt for user input - in the future, for now just use local CWD,
			// Save the files of the charts tar extracted to the local CWD
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
}
