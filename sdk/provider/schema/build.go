package schema

import (
	"encoding/json"
	"os"
)

func Build(s Schema) error {
	outputPath := os.Args[1]

	f, err := os.OpenFile(outputPath, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		return err
	}

	data, err := json.Marshal(s.ToProto())
	if err != nil {
		return err
	}

	_, err = f.Write(data)
	return err
}
