package controllers

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

// parseValuesAndQuestions parses the base64TgzChart and emits the values.yaml and questions.yaml contained within it
// If values.yaml or questions.yaml are not specified, it will return an empty string for each
func parseValuesAndQuestions(base64TgzChart string) (string, string, error) {
	tgzChartBytes, err := base64.StdEncoding.DecodeString(base64TgzChart)
	if err != nil {
		return "", "", fmt.Errorf("unable to decode base64TgzChart to tgzChart: %s", err)
	}
	gzipReader, err := gzip.NewReader(bytes.NewReader(tgzChartBytes))
	if err != nil {
		return "", "", fmt.Errorf("unable to create gzipReader to read from base64TgzChart: %s", err)
	}
	defer gzipReader.Close()
	tarReader := tar.NewReader(gzipReader)
	var valuesYamlBuffer, questionsYamlBuffer bytes.Buffer
	var foundValuesYaml, foundQuestionsYaml bool
	for {
		h, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", "", err
		}
		if h.Typeflag != tar.TypeReg {
			continue
		}
		splitName := strings.SplitN(h.Name, string(os.PathSeparator), 2)
		nameWithoutRootDir := splitName[0]
		if len(splitName) > 1 {
			nameWithoutRootDir = splitName[1]
		}
		if nameWithoutRootDir == "values.yaml" || nameWithoutRootDir == "values.yml" {
			if foundValuesYaml {
				// multiple values.yaml
				return "", "", errors.New("multiple values.yaml or values.yml found in base64TgzChart provided")
			}
			foundValuesYaml = true
			io.Copy(&valuesYamlBuffer, tarReader)
		}
		if nameWithoutRootDir == "questions.yaml" || nameWithoutRootDir == "questions.yml" {
			if foundQuestionsYaml {
				// multiple values.yaml
				return "", "", errors.New("multiple questions.yaml or questions.yml found in base64TgzChart provided")
			}
			foundQuestionsYaml = true
			io.Copy(&questionsYamlBuffer, tarReader)
		}
	}
	return valuesYamlBuffer.String(), questionsYamlBuffer.String(), nil
}
