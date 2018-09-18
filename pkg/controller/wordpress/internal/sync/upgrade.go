/*
Copyright 2018 Pressinfra SRL.

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

package sync

import (
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/presslabs/controller-util/syncer"

	wordpressv1alpha1 "github.com/presslabs/wordpress-operator/pkg/apis/wordpress/v1alpha1"
)

// NewDBUpgradeJobSyncer returns a new sync.Interface for reconciling database upgrade Job
func NewDBUpgradeJobSyncer(wp *wordpressv1alpha1.Wordpress, rt *wordpressv1alpha1.WordpressRuntime, c client.Client, scheme *runtime.Scheme) syncer.Interface {
	obj := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      wp.GetDBUpgradeJobName(rt),
			Namespace: wp.Namespace,
		},
	}

	var (
		backoffLimit          int32
		activeDeadlineSeconds int64 = 10
	)

	return syncer.NewObjectSyncer("DBUpgradeJob", wp, obj, c, scheme, func(existing runtime.Object) error {
		out := existing.(*batchv1.Job)

		if !out.CreationTimestamp.IsZero() {
			// TODO(calind): handle the case that the existing job is failed
			return nil
		}

		image := wp.GetImage(rt)
		verhash := wp.GetVersionHash(rt)
		l := wp.LabelsForComponent("db-migrate")
		l["wordpress.presslabs.org/db-upgrade-for-hash"] = verhash

		out.Labels = l
		out.Annotations = map[string]string{
			"wordpress.presslabs.org/db-upgrade-for": image,
		}

		out.Spec.BackoffLimit = &backoffLimit
		out.Spec.ActiveDeadlineSeconds = &activeDeadlineSeconds

		cmd := []string{"/bin/sh", "-c", "wp core update-db --network || wp core update-db && wp cache flush"}
		out.Spec.Template = *wp.JobPodTemplateSpec(rt, cmd...)

		out.Spec.Template.Labels = l

		return nil
	})
}
