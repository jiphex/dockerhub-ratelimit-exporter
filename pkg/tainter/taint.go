package tainter

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Tainter struct {
	cfg *rest.Config
}

const (
	taintKey = "dafni.ac.uk/dockerhub-ratelimited"
)

func (t *Tainter) client() client.Client {
	c, err := client.New(t.cfg, client.Options{})
	if err != nil {
		panic(err)
	}
	return c
}

func (t *Tainter) TaintNode(ctx context.Context, nodeName string, effect v1.TaintEffect) error {
	target := &corev1.Node{}
	err := t.client().Get(ctx, types.NamespacedName{
		Name: nodeName,
	}, target)
	patched := target.DeepCopy()
	taintWasSet := false
	for t := range patched.Spec.Taints {
		if patched.Spec.Taints[t].Key == taintKey {
			taintWasSet = true
			patched.Spec.Taints[t] = v1.Taint{
				Key:    taintKey,
				Value:  "true",
				Effect: effect,
			}
		} else {
			continue
		}
	}
	if !taintWasSet {
		patched.Spec.Taints = append(patched.Spec.Taints,
			v1.Taint{
				Key:    taintKey,
				Value:  "true",
				Effect: effect,
			},
		)
	}
	if err != nil {
		return err
	}
	err = t.client().Patch(ctx, patched, client.StrategicMergeFrom(target))
	return err
}

func NewTainter() (*Tainter, error) {
	kcfg, err := ctrl.GetConfig()
	if err != nil {
		return nil, err
	}
	t := &Tainter{
		cfg: kcfg,
	}
	return t, err
}
