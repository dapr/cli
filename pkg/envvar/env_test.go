package envvar

import (
	"bytes"
	"reflect"
	"testing"

	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
)

func secret(name, key string, optional *bool) *corev1.EnvVarSource {
	return &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{
		LocalObjectReference: corev1.LocalObjectReference{Name: name},
		Key:                  key,
		Optional:             optional,
	}}
}

// yaml.v2 has no SetIndent; its encoder already indents by 2 spaces.
func encode(t *testing.T, v interface{}) string {
	t.Helper()
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	if err := enc.Encode(v); err != nil {
		t.Fatal(err)
	}
	if err := enc.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

const sample = `
env:
  FOO: bar
  PORT: 8080
  DEBUG: true
  BAZ:
    name: mysecret
    key: password
  TOKEN:
    name: other
    key: token
    optional: true
`

func TestUnmarshal(t *testing.T) {
	var cfg Config
	if err := yaml.Unmarshal([]byte(sample), &cfg); err != nil {
		t.Fatal(err)
	}
	yes := true
	want := Env{
		{Name: "FOO", Value: "bar"},
		{Name: "PORT", Value: "8080"},
		{Name: "DEBUG", Value: "true"},
		{Name: "BAZ", ValueFrom: secret("mysecret", "password", nil)},
		{Name: "TOKEN", ValueFrom: secret("other", "token", &yes)},
	}
	if !reflect.DeepEqual(cfg.Env, want) {
		t.Fatalf("got  %#v\nwant %#v", cfg.Env, want)
	}
}

func TestMarshalGolden(t *testing.T) {
	yes := true
	got := encode(t, Config{Env: Env{
		{Name: "FOO", Value: "bar"},
		{Name: "BAZ", ValueFrom: secret("mysecret", "password", nil)},
		{Name: "TOKEN", ValueFrom: secret("other", "token", &yes)},
	}})
	want := "env:\n" +
		"  FOO: bar\n" +
		"  BAZ:\n" +
		"    name: mysecret\n" +
		"    key: password\n" +
		"  TOKEN:\n" +
		"    name: other\n" +
		"    key: token\n" +
		"    optional: true\n"
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestNumericLiteralStaysString(t *testing.T) {
	out, err := yaml.Marshal(Config{Env: Env{{Name: "PORT", Value: "8080"}}})
	if err != nil {
		t.Fatal(err)
	}
	var back Config
	if err := yaml.Unmarshal(out, &back); err != nil {
		t.Fatal(err)
	}
	if len(back.Env) != 1 || back.Env[0].Value != "8080" {
		t.Fatalf("round trip lost value: %s", out)
	}
}

func TestRoundTrip(t *testing.T) {
	var a Config
	if err := yaml.Unmarshal([]byte(sample), &a); err != nil {
		t.Fatal(err)
	}
	out, err := yaml.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	var b Config
	if err := yaml.Unmarshal(out, &b); err != nil {
		t.Fatalf("re-parse failed: %v\n%s", err, out)
	}
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("round trip mismatch\n%s", out)
	}
}

func TestOrderPreserved(t *testing.T) {
	var cfg Config
	in := "env:\n  Z: 1\n  A: 2\n  M: 3\n"
	if err := yaml.Unmarshal([]byte(in), &cfg); err != nil {
		t.Fatal(err)
	}
	got := []string{cfg.Env[0].Name, cfg.Env[1].Name, cfg.Env[2].Name}
	if !reflect.DeepEqual(got, []string{"Z", "A", "M"}) {
		t.Fatalf("order lost: %v", got)
	}
}

func TestUnmarshalErrors(t *testing.T) {
	cases := map[string]string{
		"empty mapping": "env:\n  A: {}\n",
		"missing name":  "env:\n  A:\n    key: k\n",
		"missing key":   "env:\n  A:\n    name: n\n",
		"unknown field": "env:\n  A:\n    name: n\n    key: k\n    bogus: 1\n",
		"kind":          "env:\n  A:\n    kind: configMap\n    name: n\n    key: k\n",
		"valueFrom":     "env:\n  A:\n    valueFrom:\n      secretKeyRef: {name: n, key: k}\n",
		"sequence":      "env:\n  A: [1, 2]\n",
		"not mapping":   "env: [a, b]\n",
		"duplicate":     "env:\n  A: 1\n  A: 2\n",
	}
	for name, in := range cases {
		var cfg Config
		if err := yaml.Unmarshal([]byte(in), &cfg); err == nil {
			t.Errorf("%s: expected error, got %#v", name, cfg)
		}
	}
}

func TestMarshalErrors(t *testing.T) {
	cases := map[string]Env{
		"value and from": {{Name: "A", Value: "x", ValueFrom: secret("n", "k", nil)}},
		"empty name":     {{Name: ""}},
		"duplicate":      {{Name: "A"}, {Name: "A"}},
		"no source":      {{Name: "A", ValueFrom: &corev1.EnvVarSource{}}},
		"configMap": {{Name: "A", ValueFrom: &corev1.EnvVarSource{
			ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: "n"}, Key: "k"}}}},
		"fieldRef": {{Name: "A", ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{FieldPath: "status.podIP"}}}},
		"secret and configMap": {{Name: "A", ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: secret("n", "k", nil).SecretKeyRef,
			ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: "n"}, Key: "k"}}}},
		"missing key": {{Name: "A", ValueFrom: secret("n", "", nil)}},
	}
	for name, env := range cases {
		if _, err := yaml.Marshal(Config{Env: env}); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
}
