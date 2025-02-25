package forwarder

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ViaQ/logerr/log"
	logging "github.com/openshift/cluster-logging-operator/apis/logging/v1"
	"github.com/openshift/cluster-logging-operator/internal/generator"
	"github.com/openshift/cluster-logging-operator/internal/generator/fluentd"
	"github.com/openshift/cluster-logging-operator/internal/generator/vector"
	corev1 "k8s.io/api/core/v1"
	"net/url"
)

var (
	ErrNoOutputs        = errors.New("No outputs defined in ClusterLogForwarder")
	ErrNoValidInputs    = errors.New("No valid inputs found in ClusterLogForwarder")
	ErrInvalidOutputURL = func(o logging.OutputSpec) error {
		return fmt.Errorf("Invalid URL in %s output in ClusterLogForwarder", o.Name)
	}
	ErrInvalidInput = errors.New("Invalid Input")
)

type ConfigGenerator struct {
	g    generator.Generator
	conf func(clspec *logging.ClusterLoggingSpec, secrets map[string]*corev1.Secret, clfspec *logging.ClusterLogForwarderSpec, op generator.Options) []generator.Section
}

func New(collectorType logging.LogCollectionType) *ConfigGenerator {
	g := &ConfigGenerator{}
	switch collectorType {
	case logging.LogCollectionTypeFluentd:
		g.conf = fluentd.Conf
	case logging.LogCollectionTypeVector:
		g.conf = vector.Conf
	default:
		log.Error(errors.New("Unsupported collector implementation"), "type", collectorType)
		return nil
	}
	return g
}

func (cg *ConfigGenerator) GenerateConf(clspec *logging.ClusterLoggingSpec, secrets map[string]*corev1.Secret, clfspec *logging.ClusterLogForwarderSpec, op generator.Options) (string, error) {
	sections := cg.conf(clspec, secrets, clfspec, op)
	return cg.g.GenerateConf(generator.MergeSections(sections)...)
}

func (cg *ConfigGenerator) Verify(clspec *logging.ClusterLoggingSpec, secrets map[string]*corev1.Secret, clfspec *logging.ClusterLogForwarderSpec, op generator.Options) error {
	var err error
	types := generator.GatherSources(clfspec, op)
	if !types.HasAny(logging.InputNameApplication, logging.InputNameInfrastructure, logging.InputNameAudit) {
		return ErrNoValidInputs
	}
	if len(clfspec.Outputs) == 0 {
		return ErrNoOutputs
	}
	for _, p := range clfspec.Pipelines {
		if _, err := json.Marshal(p.Labels); err != nil {
			return ErrInvalidInput
		}
	}
	for _, o := range clfspec.Outputs {
		if _, err := url.Parse(o.URL); err != nil {
			return ErrInvalidOutputURL(o)
		}
	}
	return err
}
