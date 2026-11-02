package envvar

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
)

type Env []corev1.EnvVar

type Config struct {
	Env Env `yaml:"env,omitempty"`
}

func (e *Env) UnmarshalYAML(n *yaml.Node) error {
	for n.Kind == yaml.AliasNode {
		n = n.Alias
	}
	if n.ShortTag() == "!!null" {
		*e = nil
		return nil
	}
	if n.Kind != yaml.MappingNode {
		return fmt.Errorf("line %d: env must be a mapping of NAME to value", n.Line)
	}

	out := make(Env, 0, len(n.Content)/2)
	seen := make(map[string]struct{}, len(n.Content)/2)
	for i := 0; i+1 < len(n.Content); i += 2 {
		k, v := n.Content[i], n.Content[i+1]
		if k.Kind != yaml.ScalarNode || k.Value == "" {
			return fmt.Errorf("line %d: env variable name must be a non-empty string", k.Line)
		}
		if _, dup := seen[k.Value]; dup {
			return fmt.Errorf("line %d: duplicate env variable %q", k.Line, k.Value)
		}
		seen[k.Value] = struct{}{}

		ev, err := decodeEnvVar(k.Value, v)
		if err != nil {
			return fmt.Errorf("env %q: %w", k.Value, err)
		}
		out = append(out, ev)
	}
	*e = out
	return nil
}

func decodeEnvVar(name string, v *yaml.Node) (corev1.EnvVar, error) {
	for v.Kind == yaml.AliasNode {
		v = v.Alias
	}
	ev := corev1.EnvVar{Name: name}

	switch v.Kind {
	case yaml.ScalarNode:
		if v.ShortTag() != "!!null" {
			ev.Value = v.Value
		}
		return ev, nil

	case yaml.MappingNode:
		sel, err := decodeSecretKeySelector(v)
		if err != nil {
			return ev, err
		}
		ev.ValueFrom = &corev1.EnvVarSource{SecretKeyRef: sel}
		return ev, nil

	default:
		return ev, fmt.Errorf("line %d: must be a scalar or a mapping", v.Line)
	}
}

func decodeSecretKeySelector(n *yaml.Node) (*corev1.SecretKeySelector, error) {
	var raw interface{}
	if err := n.Decode(&raw); err != nil { // also rejects duplicate keys
		return nil, err
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("line %d: %w", n.Line, err)
	}

	var sel corev1.SecretKeySelector
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&sel); err != nil {
		return nil, fmt.Errorf("line %d: %w", n.Line, err)
	}
	if sel.Name == "" || sel.Key == "" {
		return nil, fmt.Errorf("line %d: name and key are required", n.Line)
	}
	return &sel, nil
}

func (e Env) MarshalYAML() (interface{}, error) {
	m := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	seen := make(map[string]struct{}, len(e))

	for _, ev := range e {
		if ev.Name == "" {
			return nil, fmt.Errorf("env variable with empty name")
		}
		if _, dup := seen[ev.Name]; dup {
			return nil, fmt.Errorf("duplicate env variable %q", ev.Name)
		}
		seen[ev.Name] = struct{}{}

		key := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: ev.Name}
		var val *yaml.Node

		if ev.ValueFrom == nil {
			val = &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: ev.Value}
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
			var err error
			if val, err = selectorToNode(s.SecretKeyRef); err != nil {
				return nil, fmt.Errorf("env %q: %w", ev.Name, err)
			}
		}
		m.Content = append(m.Content, key, val)
	}
	return m, nil
}

func selectorToNode(sel *corev1.SecretKeySelector) (*yaml.Node, error) {
	b, err := json.Marshal(sel)
	if err != nil {
		return nil, err
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	return jsonToNode(dec)
}

func jsonToNode(dec *json.Decoder) (*yaml.Node, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	switch t := tok.(type) {
	case json.Delim:
		switch t {
		case '{':
			n := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
			for dec.More() {
				kt, err := dec.Token()
				if err != nil {
					return nil, err
				}
				ks, ok := kt.(string)
				if !ok {
					return nil, fmt.Errorf("unexpected JSON object key %v", kt)
				}
				v, err := jsonToNode(dec)
				if err != nil {
					return nil, err
				}
				n.Content = append(n.Content,
					&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: ks}, v)
			}
			if _, err := dec.Token(); err != nil { // closing '}'
				return nil, err
			}
			return n, nil
		case '[':
			n := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
			for dec.More() {
				v, err := jsonToNode(dec)
				if err != nil {
					return nil, err
				}
				n.Content = append(n.Content, v)
			}
			if _, err := dec.Token(); err != nil { // closing ']'
				return nil, err
			}
			return n, nil
		}
		return nil, fmt.Errorf("unexpected JSON delimiter %v", t)
	case string:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: t}, nil
	case json.Number:
		tag := "!!int"
		if strings.ContainsAny(t.String(), ".eE") {
			tag = "!!float"
		}
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: tag, Value: t.String()}, nil
	case bool:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: strconv.FormatBool(t)}, nil
	case nil:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!null", Value: "null"}, nil
	}
	return nil, fmt.Errorf("unexpected JSON token %v", tok)
}
