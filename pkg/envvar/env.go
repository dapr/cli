package envvar

import (
	"bytes"
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
)

type Env []corev1.EnvVar

type Config struct {
	Env Env `yaml:"env,omitempty"`
}

// envValue captures a single env entry: either a scalar (kept as its raw
// text) or a secretKeyRef-style mapping.
type envValue struct {
	scalar string
	sel    *corev1.SecretKeySelector
}

func (v *envValue) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// Decoding any resolved scalar (int, bool, float, str) into a string
	// keeps its original text; null yields "".
	var s string
	if err := unmarshal(&s); err == nil {
		v.scalar = s
		return nil
	}

	var m map[string]interface{}
	if err := unmarshal(&m); err != nil {
		return fmt.Errorf("must be a scalar or a mapping")
	}
	sel, err := decodeSecretKeySelector(m)
	if err != nil {
		return err
	}
	v.sel = sel
	return nil
}

func (e *Env) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// First pass: ordered keys (detects duplicates and non-string keys).
	var ordered yaml.MapSlice
	if err := unmarshal(&ordered); err != nil {
		return fmt.Errorf("env must be a mapping of NAME to value: %w", err)
	}
	if ordered == nil {
		*e = nil
		return nil
	}

	// Second pass: values, preserving raw scalar text.
	var values map[string]envValue
	if err := unmarshal(&values); err != nil {
		return err
	}

	out := make(Env, 0, len(ordered))
	seen := make(map[string]struct{}, len(ordered))
	for _, item := range ordered {
		name, ok := item.Key.(string)
		if !ok || name == "" {
			return fmt.Errorf("env variable name must be a non-empty string, got %v", item.Key)
		}
		if _, dup := seen[name]; dup {
			return fmt.Errorf("duplicate env variable %q", name)
		}
		seen[name] = struct{}{}

		v := values[name]
		ev := corev1.EnvVar{Name: name}
		if v.sel != nil {
			ev.ValueFrom = &corev1.EnvVarSource{SecretKeyRef: v.sel}
		} else {
			ev.Value = v.scalar
		}
		out = append(out, ev)
	}
	*e = out
	return nil
}

func decodeSecretKeySelector(m map[string]interface{}) (*corev1.SecretKeySelector, error) {
	b, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}

	var sel corev1.SecretKeySelector
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&sel); err != nil {
		return nil, err
	}
	if sel.Name == "" || sel.Key == "" {
		return nil, fmt.Errorf("name and key are required")
	}
	return &sel, nil
}

func (e Env) MarshalYAML() (interface{}, error) {
	m := make(yaml.MapSlice, 0, len(e))
	seen := make(map[string]struct{}, len(e))

	for _, ev := range e {
		if ev.Name == "" {
			return nil, fmt.Errorf("env variable with empty name")
		}
		if _, dup := seen[ev.Name]; dup {
			return nil, fmt.Errorf("duplicate env variable %q", ev.Name)
		}
		seen[ev.Name] = struct{}{}

		var val interface{}
		if ev.ValueFrom == nil {
			val = ev.Value
		} else {
			if ev.Value != "" {
				return nil, fmt.Errorf("env %q: cannot set both Value and ValueFrom", ev.Name)
			}
			s := ev.ValueFrom
			if s.SecretKeyRef == nil || s.FieldRef != nil || s.ResourceFieldRef != nil || s.ConfigMapKeyRef != nil {
				return nil, fmt.Errorf("env %q: only secretKeyRef is supported", ev.Name)
			}
			if s.SecretKeyRef.Name == "" || s.SecretKeyRef.Key == "" {
				return nil, fmt.Errorf("env %q: secretKeyRef requires name and key", ev.Name)
			}
			val = selectorToMapSlice(s.SecretKeyRef)
		}
		m = append(m, yaml.MapItem{Key: ev.Name, Value: val})
	}
	return m, nil
}

func selectorToMapSlice(sel *corev1.SecretKeySelector) yaml.MapSlice {
	out := yaml.MapSlice{
		{Key: "name", Value: sel.Name},
		{Key: "key", Value: sel.Key},
	}
	if sel.Optional != nil {
		out = append(out, yaml.MapItem{Key: "optional", Value: *sel.Optional})
	}
	return out
}
