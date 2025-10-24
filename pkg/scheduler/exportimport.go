/*
Copyright 2025 The Dapr Authors
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package scheduler

import (
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"os"

	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/protobuf/proto"

	"github.com/dapr/cli/pkg/print"
	"github.com/dapr/cli/pkg/scheduler/stored"
)

type ExportImportOptions struct {
	SchedulerNamespace string
	KubernetesMode     bool
	TargetFile         string
}

type ExportFile struct {
	Jobs     map[string][]byte
	Counters map[string][]byte
}

func Export(ctx context.Context, opts ExportImportOptions) error {
	if _, err := os.Stat(opts.TargetFile); !errors.Is(err, os.ErrNotExist) {
		if err == nil {
			return fmt.Errorf("file '%s' already exists", opts.TargetFile)
		}
		return err
	}

	client, cancel, err := etcdClient(opts.KubernetesMode, opts.SchedulerNamespace)
	if err != nil {
		return err
	}
	defer cancel()

	jobs, err := listJobs(ctx, client)
	if err != nil {
		return err
	}
	counters, err := listCounters(ctx, client)
	if err != nil {
		return err
	}

	out := ExportFile{
		Jobs:     make(map[string][]byte, len(jobs)),
		Counters: make(map[string][]byte, len(counters)),
	}

	var b []byte
	for k, j := range jobs {
		b, err = proto.Marshal(j)
		if err != nil {
			return fmt.Errorf("marshal job %q: %w", k, err)
		}
		out.Jobs[k] = b
	}
	for k, c := range counters {
		b, err = proto.Marshal(c)
		if err != nil {
			return fmt.Errorf("marshal counter %q: %w", k, err)
		}
		out.Counters[k] = b
	}

	f, err := os.OpenFile(opts.TargetFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("open %s: %w", opts.TargetFile, err)
	}
	defer f.Close()

	if err := gob.NewEncoder(f).Encode(&out); err != nil {
		_ = os.Remove(opts.TargetFile)
		return fmt.Errorf("encode export file: %w", err)
	}

	print.InfoStatusEvent(os.Stdout, "Exported %d jobs and %d counters.", len(out.Jobs), len(out.Counters))
	return nil
}

func Import(ctx context.Context, opts ExportImportOptions) error {
	client, cancel, err := etcdClient(opts.KubernetesMode, opts.SchedulerNamespace)
	if err != nil {
		return err
	}
	defer cancel()

	f, err := os.OpenFile(opts.TargetFile, os.O_RDONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open %s: %w", opts.TargetFile, err)
	}
	defer f.Close()

	var in ExportFile
	if err := gob.NewDecoder(f).Decode(&in); err != nil {
		return fmt.Errorf("decode import file: %w", err)
	}

	ops := make([]clientv3.Op, 0, len(in.Jobs)+len(in.Counters))

	for key, b := range in.Jobs {
		var j stored.Job
		if err := proto.Unmarshal(b, &j); err != nil {
			return fmt.Errorf("unmarshal job %q: %w", key, err)
		}
		ops = append(ops, clientv3.OpPut(key, string(b)))
	}

	for key, b := range in.Counters {
		var c stored.Counter
		if err := proto.Unmarshal(b, &c); err != nil {
			return fmt.Errorf("unmarshal counter %q: %w", key, err)
		}
		ops = append(ops, clientv3.OpPut(key, string(b)))
	}

	var end int
	for i := 0; i < len(ops); i += 128 {
		txn := client.Txn(ctx)
		end = i + 128
		if end > len(ops) {
			end = len(ops)
		}
		txn.Then(ops[i:end]...)
		if _, err := txn.Commit(); err != nil {
			print.FailureStatusEvent(os.Stderr, "Incomplete import with %d items.", end)
			return fmt.Errorf("commit transaction: %w", err)
		}
	}

	print.InfoStatusEvent(os.Stdout, "Imported %d items.", end)

	return nil
}
