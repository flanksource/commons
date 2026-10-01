package properties

import "sort"

type PropertyType string

const (
	PropertyTypeString        PropertyType = "string"
	PropertyTypeBool          PropertyType = "bool"
	PropertyTypeInt           PropertyType = "int"
	PropertyTypeBytes         PropertyType = "bytes"
	PropertyTypeDuration      PropertyType = "duration"
	PropertyTypeChoice        PropertyType = "choice"
	PropertyTypeLogLevel      PropertyType = "log-level"
	PropertyTypeTimeIntervals PropertyType = "time-intervals"
)

type PropertySource string

const (
	PropertySourceDefault     PropertySource = "default"
	PropertySourceRuntime     PropertySource = "runtime"
	PropertySourceCommandLine PropertySource = "command-line"
	PropertySourceUnset       PropertySource = "unset"
)

type Property struct {
	Key        string
	Value      string
	Default    string
	HasDefault bool
	Type       PropertyType
	Options    []string
	Source     PropertySource
	ReadOnly   bool
}

type propertyMetadata struct {
	Default    string
	HasDefault bool
	Type       PropertyType
	Options    []string
}

func (p *Properties) registerDefault(valueType PropertyType, defaultValue string, keys ...string) {
	metadata := propertyMetadata{Default: defaultValue, HasDefault: true, Type: valueType}
	for _, key := range keys {
		p.metadata.Store(key, metadata)
	}
}

func (p *Properties) registerChoice(valueType PropertyType, defaultValue string, options []string, keys ...string) {
	metadata := propertyMetadata{
		Default: defaultValue, HasDefault: true, Type: valueType, Options: append([]string(nil), options...),
	}
	for _, key := range keys {
		p.metadata.Store(key, metadata)
	}
}

func (p *Properties) registerType(valueType PropertyType, keys ...string) {
	metadata := propertyMetadata{Type: valueType}
	for _, key := range keys {
		p.metadata.Store(key, metadata)
	}
}

func (p *Properties) List() []Property {
	p.lock.RLock()
	defer p.lock.RUnlock()

	keys := make(map[string]struct{}, len(p.m)+len(commandlineProperties))
	for key := range p.m {
		keys[key] = struct{}{}
	}
	p.metadata.Range(func(key, _ any) bool {
		keys[key.(string)] = struct{}{}
		return true
	})
	for key := range commandlineProperties {
		keys[key] = struct{}{}
	}

	sortedKeys := make([]string, 0, len(keys))
	for key := range keys {
		sortedKeys = append(sortedKeys, key)
	}
	sort.Strings(sortedKeys)

	result := make([]Property, 0, len(sortedKeys))
	for _, key := range sortedKeys {
		metadataValue, declared := p.metadata.Load(key)
		metadata, _ := metadataValue.(propertyMetadata)
		property := Property{Key: key, Type: PropertyTypeString, Source: PropertySourceUnset}
		if declared {
			property.Default = metadata.Default
			property.HasDefault = metadata.HasDefault
			property.Type = metadata.Type
			property.Options = append([]string(nil), metadata.Options...)
		}
		if value, overridden := commandlineProperties[key]; overridden {
			if value == "" && metadata.HasDefault {
				property.Value = metadata.Default
			} else {
				property.Value = value
			}
			property.Source = PropertySourceCommandLine
			property.ReadOnly = true
		} else if value, configured := p.m[key]; configured && (value != "" || !metadata.HasDefault) {
			property.Value = value
			property.Source = PropertySourceRuntime
		} else if metadata.HasDefault {
			property.Value = metadata.Default
			property.Source = PropertySourceDefault
		}
		result = append(result, property)
	}
	return result
}

func List() []Property {
	return Global.List()
}
