package lookup

import (
	"strconv"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func Set(object interface{}, key, val string) error {
	log.Debugf("Looking up %s to set it to: %s", key, val)

	value, err := LookupString(object, key)
	if err != nil {
		return errors.Wrapf(err, "cannot lookup %s", key)
	}
	log.Infof("Overriding %s %v => %v", key, value, val)
	switch value.Interface().(type) {
	case string:
		value.SetString(val)
	case int:
		i, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return errors.Wrapf(err, "cannot convert %s to int", val)
		}
		value.SetInt(i)
	case bool:
		b, err := strconv.ParseBool(val)
		if err != nil {
			return errors.Wrapf(err, "cannot convert %s to boolean", val)
		}
		value.SetBool(b)
	}

	return nil
}
