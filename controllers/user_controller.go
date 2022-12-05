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
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"reflect"
	"sort"
	"strconv"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	usersv1alpha1 "github.com/adrafiq/reqres-controller/api/v1alpha1"
)

// USERReconciler reconciles a USER object
type USERReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

type UserCreateResponse struct {
	Id        string `json:"id"`
	CreatedAt string `json:"createdAt"`
}

type UserGetResponse struct {
	Data struct {
		Id        int    `json:"id"`
		Email     string `json:"email"`
		FirstName string `json:"first_name,omitempty"`
		LastName  string `json:"last_name,omitempty"`
		Avatar    string `json:"avatar,omitempty"`
	} `json:"data"`
	Support struct{} `json:"support,omitempty"`
}

const (
	notInitialized    = 0
	httpPostSuccess   = 201
	httpGetSuccess    = 200
	httpDeleteSuccess = 204
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
	var userStatus usersv1alpha1.USERStatus
	err := r.Get(ctx, req.NamespacedName, userCR)
	if err != nil && errors.IsNotFound(err) {
		//send http delete
		logger.Info("Object Deleted")
		return ctrl.Result{}, nil
	} else if err != nil {
		logger.Error(err, "Error getting operator resource object")
		return ctrl.Result{}, err
	}

	client := http.DefaultClient
	// If deleted, http delete and remove finalizer
	if userCR.ObjectMeta.DeletionTimestamp != nil {
		api := `api/users/` + strconv.Itoa(userCR.Status.Id)
		url := "https://reqres.in/" + api
		httpReq, _ := http.NewRequest("DELETE", url, nil)
		res, err := client.Do(httpReq)
		if err != nil {
			logger.Error(err, "error making http request")
			return ctrl.Result{Requeue: true}, nil
		}
		defer res.Body.Close()
		if res.StatusCode == httpDeleteSuccess {
			finalizers := userCR.ObjectMeta.Finalizers
			ctrlIndex := sort.SearchStrings(finalizers, "reqres")
			finalizers = append(finalizers[:ctrlIndex], finalizers[ctrlIndex+1:]...)
			userCR.Finalizers = finalizers
		}
		r.Update(ctx, userCR)
		return ctrl.Result{}, nil
	}

	// Create user in backend, if not exists
	if userCR.Status.Id == notInitialized {
		postBody, _ := json.Marshal(map[string]string{
			"email":      userCR.Spec.Email,
			"first_name": userCR.Spec.FirstName,
			"last_name":  userCR.Spec.LastName,
		})
		body := bytes.NewBuffer(postBody)
		api := `api/users/`
		url := "https://reqres.in/" + api
		httpReq, _ := http.NewRequest("POST", url, body)
		res, err := client.Do(httpReq)
		if err != nil {
			return ctrl.Result{Requeue: true}, nil
		}
		defer res.Body.Close()
		if res.StatusCode == httpPostSuccess {
			var response UserCreateResponse
			resBody, _ := ioutil.ReadAll(res.Body)
			json.Unmarshal(resBody, &response)
			id, _ := strconv.Atoi(response.Id)
			userStatus = usersv1alpha1.USERStatus{
				Id: id,
				Conditions: []metav1.Condition{{
					Type:               "Available",
					Status:             metav1.ConditionTrue,
					LastTransitionTime: metav1.NewTime(time.Now()),
					Reason:             "OperatorSucceeded",
					Message:            "user successfully created",
				}},
			}
		}
	} else {
		// Check if CR not equals to backend obj, update it
		api := `api/users/` + strconv.Itoa(userCR.Status.Id)
		url := "https://reqres.in/" + api
		httpReq, _ := http.NewRequest("GET", url, nil)
		res, err := client.Do(httpReq)
		if err != nil {
			logger.Error(err, "error making http request")
			return ctrl.Result{Requeue: true}, nil
		}
		defer res.Body.Close()
		if res.StatusCode != httpGetSuccess {
			logger.Error(nil, "unable to find user in backend")
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
		resBody, _ := ioutil.ReadAll(res.Body)
		var userGetResponse UserGetResponse
		json.Unmarshal(resBody, &userGetResponse)
		user := usersv1alpha1.USERSpec{
			Email:     userGetResponse.Data.Email,
			FirstName: userGetResponse.Data.FirstName,
			LastName:  userGetResponse.Data.LastName,
			Avatar:    userGetResponse.Data.Avatar,
		}
		userStatus = usersv1alpha1.USERStatus{
			Id: userGetResponse.Data.Id,
			Conditions: []metav1.Condition{{
				Type:               "Available",
				Status:             metav1.ConditionTrue,
				LastTransitionTime: metav1.NewTime(time.Now()),
				Reason:             "OperatorSucceeded",
				Message:            "user successfully synced",
			}},
		}
		if !reflect.DeepEqual(user, userCR.Spec) {
			userCR.Spec = user
			// Patch User
			client := http.DefaultClient
			postBody, _ := json.Marshal(map[string]string{
				"email":      userCR.Spec.Email,
				"first_name": userCR.Spec.FirstName,
				"last_name":  userCR.Spec.LastName,
			})
			body := bytes.NewBuffer(postBody)
			api := `api/users/` + strconv.Itoa(userCR.Status.Id)
			url := "https://reqres.in/" + api
			httpReq, _ := http.NewRequest("PATCH", url, body)
			res, err := client.Do(httpReq)
			if err != nil {
				return ctrl.Result{Requeue: true}, nil
			}
			defer res.Body.Close()
			userStatus = usersv1alpha1.USERStatus{
				Id: userGetResponse.Data.Id,
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
