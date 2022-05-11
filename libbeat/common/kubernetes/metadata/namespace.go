// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package metadata

import (
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/elastic/beats/v7/libbeat/common/kubernetes"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

const resource = "namespace"

type namespace struct {
	store    cache.Store
	resource *Resource
}

// NewNamespaceMetadataGenerator creates a metagen for namespace resources
func NewNamespaceMetadataGenerator(cfg *config.C, namespaces cache.Store, client k8s.Interface) MetaGen {
	return &namespace{
		resource: NewResourceMetadataGenerator(cfg, client),
		store:    namespaces,
	}
}

// Generate generates pod metadata from a resource object
// Metadata map is in the following form:
// {
// 	  "kubernetes": {},
//    "some.ecs.field": "asdf"
// }
// All Kubernetes fields that need to be stored under kuberentes. prefix are populetad by
// GenerateK8s method while fields that are part of ECS are generated by GenerateECS method
func (n *namespace) Generate(obj kubernetes.Resource, opts ...FieldOptions) mapstr.M {
	ecsFields := n.GenerateECS(obj)
	meta := mapstr.M{
		"kubernetes": n.GenerateK8s(obj, opts...),
	}
	meta.DeepUpdate(ecsFields)
	return meta
}

// GenerateECS generates namespace ECS metadata from a resource object
func (n *namespace) GenerateECS(obj kubernetes.Resource) mapstr.M {
	return n.resource.GenerateECS(obj)
}

// GenerateK8s generates namespace metadata from a resource object
func (n *namespace) GenerateK8s(obj kubernetes.Resource, opts ...FieldOptions) mapstr.M {
	_, ok := obj.(*kubernetes.Namespace)
	if !ok {
		return nil
	}

	meta := n.resource.GenerateK8s(resource, obj, opts...)
	meta = flattenMetadata(meta)

	// TODO: Add extra fields in here if need be
	return meta
}

// GenerateFromName generates pod metadata from a namespace name
func (n *namespace) GenerateFromName(name string, opts ...FieldOptions) mapstr.M {
	if n.store == nil {
		return nil
	}

	if obj, ok, _ := n.store.GetByKey(name); ok {
		no, ok := obj.(*kubernetes.Namespace)
		if !ok {
			return nil
		}

		return n.GenerateK8s(no, opts...)
	}

	return nil
}

func flattenMetadata(in mapstr.M) mapstr.M {
	out := mapstr.M{}
	rawFields, err := in.GetValue(resource)
	if err != nil {
		return nil
	}

	fields, ok := rawFields.(mapstr.M)
	if !ok {
		return nil
	}
	for k, v := range fields {
		if k == "name" {
			out[resource] = v
		} else {
			out[resource+"_"+k] = v
		}
	}

	populateFromKeys := []string{"labels", "annotations"}
	for _, key := range populateFromKeys {
		rawValues, err := in.GetValue(key)
		if err != nil {
			continue
		}
		values, ok := rawValues.(mapstr.M)
		if ok {
			out[resource+"_"+key] = values
		}
	}

	return out
}
