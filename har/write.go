package har

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
)

// CreatorName identifies commons as the producer in the HAR envelope.
const CreatorName = "flanksource-commons"

// WriteFile serializes collector.Entries() into a HAR 1.2 file at path. A
// collector configured with CaptureSensitive holds unmasked credentials, so its
// file is written 0600 rather than 0644.
func WriteFile(collector *Collector, path string) error {
	file := File{
		Log: Log{
			Version: "1.2",
			Creator: Creator{Name: CreatorName, Version: "0"},
			Pages:   []Page{},
			Entries: collector.Entries(),
		},
	}
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal HAR: %w", err)
	}
	return os.WriteFile(path, append(data, '\n'), fileMode(collector))
}

func fileMode(collector *Collector) fs.FileMode {
	if collector != nil && collector.Config.CaptureSensitive {
		return 0o600
	}
	return 0o644
}
