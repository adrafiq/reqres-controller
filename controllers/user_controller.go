/*
Copyright 2022.

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

package controllers

import (
	"context"
	"reflect"
	"sort"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	usersv1alpha1 "github.com/adrafiq/reqres-controller/api/v1alpha1"
	reqres "github.com/adrafiq/reqres-controller/pkg/reqres"
	"github.com/spf13/viper"
)

// USERReconciler reconciles a USER object
type USERReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Config *viper.Viper
}

const (
	notInitialized = 0
	ctrlFinalizer  = "users.reqres.in/v1alpha1"
)

//+kubebuilder:rbac:groups=users.reqres.in,resources=users,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=users.reqres.in,resources=users/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=users.reqres.in,resources=users/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the USER object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *USERReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	userCR := &usersv1alpha1.USER{}
	reqresURL := r.Config.GetString("REQRES_ROOT_URL")
	client := reqres.NewClient(reqresURL, &logger)
	var userStatus usersv1alpha1.USERStatus
	err := r.Get(ctx, req.NamespacedName, userCR)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Object Deleted")
		return ctrl.Result{}, nil
	} else if err != nil {
		logger.Error(err, "Error getting operator resource object")
		return ctrl.Result{}, err
	}

	// If deleted, http delete and remove finalizer
	if userCR.ObjectMeta.DeletionTimestamp != nil {
		client.DeleteUser(userCR.Status.Id)
		if err != nil {
			logger.Error(err, "http client error")
			return ctrl.Result{Requeue: true}, nil
		}
		finalizers := userCR.ObjectMeta.Finalizers
		idx := sort.SearchStrings(finalizers, ctrlFinalizer)
		finalizers = append(finalizers[:idx], finalizers[idx+1:]...)
		userCR.Finalizers = finalizers
		r.Update(ctx, userCR)
		return ctrl.Result{}, nil
	}

	// Create user in backend, if not exists
	if userCR.Status.Id == notInitialized {
		user := reqres.User{
			Email:     userCR.Spec.Email,
			FirstName: userCR.Spec.FirstName,
			LastName:  userCR.Spec.LastName,
		}
		userCreated, err := client.CreateUser(user)
		if err != nil {
			logger.Error(err, "http client error")
			return ctrl.Result{Requeue: true}, nil
		}
		userStatus = usersv1alpha1.USERStatus{
			Id: userCreated.Id,
			Conditions: []metav1.Condition{{
				Type:               "Available",
				Status:             metav1.ConditionTrue,
				LastTransitionTime: metav1.NewTime(time.Now()),
				Reason:             "OperatorSucceeded",
				Message:            "user successfully created",
			}},
		}
	} else {
		// Check if CR not equals to backend obj, update it
		user, err := client.GetUser(userCR.Status.Id)
		if err != nil && err.Error() == "error making http request" {
			logger.Error(err, "http client error")
			return ctrl.Result{Requeue: true}, nil
		} else if err != nil {
			logger.Error(err, "unable to find user in backend")
			userStatus = usersv1alpha1.USERStatus{
				Id: 0,
				Conditions: []metav1.Condition{{
					Type:               "Unavailable",
					Status:             metav1.ConditionUnknown,
					LastTransitionTime: metav1.NewTime(time.Now()),
					Reason:             "OperatorSucceeded",
					Message:            "could not find user in backend",
				}},
			}
			userCR.Status = userStatus
			if err := r.Status().Update(ctx, userCR); err != nil {
				logger.Info("unable to update status")
			}
			return ctrl.Result{Requeue: true}, nil
		}
		userStatus = usersv1alpha1.USERStatus{
			Id: user.Id,
			Conditions: []metav1.Condition{{
				Type:               "Available",
				Status:             metav1.ConditionTrue,
				LastTransitionTime: metav1.NewTime(time.Now()),
				Reason:             "OperatorSucceeded",
				Message:            "user successfully synced",
			}},
		}
		userFromCR := reqres.User{
			Email:     userCR.Spec.Email,
			FirstName: userCR.Spec.FirstName,
			LastName:  userCR.Spec.LastName,
		}
		if !reflect.DeepEqual(user, userFromCR) {
			// Patch User
			err := client.UpdateUser(*&userFromCR)
			if err != nil {
				logger.Error(err, "error making http request")
				return ctrl.Result{Requeue: true}, nil
			}
			userStatus = usersv1alpha1.USERStatus{
				Id: user.Id,
				Conditions: []metav1.Condition{{
					Type:               "Available",
					Status:             metav1.ConditionTrue,
					LastTransitionTime: metav1.NewTime(time.Now()),
					Reason:             "OperatorSucceeded",
					Message:            "user successfully updated",
				}},
			}
		}
	}

	userCR.Status = userStatus
	if err := r.Status().Update(ctx, userCR); err != nil {
		logger.Info("unable to update status")
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *USERReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&usersv1alpha1.USER{}).
		Complete(r)
}
