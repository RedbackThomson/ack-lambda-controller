// Copyright Amazon.com Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may
// not use this file except in compliance with the License. A copy of the
// License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing
// permissions and limitations under the License.

// Code generated by ack-generate. DO NOT EDIT.

package function

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	ec2apitypes "github.com/aws-controllers-k8s/ec2-controller/apis/v1alpha1"
	kmsapitypes "github.com/aws-controllers-k8s/kms-controller/apis/v1alpha1"
	ackv1alpha1 "github.com/aws-controllers-k8s/runtime/apis/core/v1alpha1"
	ackcondition "github.com/aws-controllers-k8s/runtime/pkg/condition"
	ackerr "github.com/aws-controllers-k8s/runtime/pkg/errors"
	acktypes "github.com/aws-controllers-k8s/runtime/pkg/types"

	svcapitypes "github.com/aws-controllers-k8s/lambda-controller/apis/v1alpha1"
)

// +kubebuilder:rbac:groups=kms.services.k8s.aws,resources=keys,verbs=get;list
// +kubebuilder:rbac:groups=kms.services.k8s.aws,resources=keys/status,verbs=get;list

// +kubebuilder:rbac:groups=ec2.services.k8s.aws,resources=securitygroups,verbs=get;list
// +kubebuilder:rbac:groups=ec2.services.k8s.aws,resources=securitygroups/status,verbs=get;list

// +kubebuilder:rbac:groups=ec2.services.k8s.aws,resources=subnets,verbs=get;list
// +kubebuilder:rbac:groups=ec2.services.k8s.aws,resources=subnets/status,verbs=get;list

// ResolveReferences finds if there are any Reference field(s) present
// inside AWSResource passed in the parameter and attempts to resolve
// those reference field(s) into target field(s).
// It returns an AWSResource with resolved reference(s), and an error if the
// passed AWSResource's reference field(s) cannot be resolved.
// This method also adds/updates the ConditionTypeReferencesResolved for the
// AWSResource.
func (rm *resourceManager) ResolveReferences(
	ctx context.Context,
	apiReader client.Reader,
	res acktypes.AWSResource,
) (acktypes.AWSResource, error) {
	namespace := res.MetaObject().GetNamespace()
	ko := rm.concreteResource(res).ko.DeepCopy()
	err := validateReferenceFields(ko)
	if err == nil {
		err = resolveReferenceForKMSKeyARN(ctx, apiReader, namespace, ko)
	}
	if err == nil {
		err = resolveReferenceForVPCConfig_SecurityGroupIDs(ctx, apiReader, namespace, ko)
	}
	if err == nil {
		err = resolveReferenceForVPCConfig_SubnetIDs(ctx, apiReader, namespace, ko)
	}

	// If there was an error while resolving any reference, reset all the
	// resolved values so that they do not get persisted inside etcd
	if err != nil {
		ko = rm.concreteResource(res).ko.DeepCopy()
	}
	if hasNonNilReferences(ko) {
		return ackcondition.WithReferencesResolvedCondition(&resource{ko}, err)
	}
	return &resource{ko}, err
}

// validateReferenceFields validates the reference field and corresponding
// identifier field.
func validateReferenceFields(ko *svcapitypes.Function) error {
	if ko.Spec.KMSKeyRef != nil && ko.Spec.KMSKeyARN != nil {
		return ackerr.ResourceReferenceAndIDNotSupportedFor("KMSKeyARN", "KMSKeyRef")
	}
	if ko.Spec.VPCConfig != nil {
		if ko.Spec.VPCConfig.SecurityGroupRefs != nil && ko.Spec.VPCConfig.SecurityGroupIDs != nil {
			return ackerr.ResourceReferenceAndIDNotSupportedFor("VPCConfig.SecurityGroupIDs", "VPCConfig.SecurityGroupRefs")
		}
	}
	if ko.Spec.VPCConfig != nil {
		if ko.Spec.VPCConfig.SubnetRefs != nil && ko.Spec.VPCConfig.SubnetIDs != nil {
			return ackerr.ResourceReferenceAndIDNotSupportedFor("VPCConfig.SubnetIDs", "VPCConfig.SubnetRefs")
		}
	}
	return nil
}

// hasNonNilReferences returns true if resource contains a reference to another
// resource
func hasNonNilReferences(ko *svcapitypes.Function) bool {
	return false || (ko.Spec.KMSKeyRef != nil) || (ko.Spec.VPCConfig != nil && ko.Spec.VPCConfig.SecurityGroupRefs != nil) || (ko.Spec.VPCConfig != nil && ko.Spec.VPCConfig.SubnetRefs != nil)
}

// resolveReferenceForKMSKeyARN reads the resource referenced
// from KMSKeyRef field and sets the KMSKeyARN
// from referenced resource
func resolveReferenceForKMSKeyARN(
	ctx context.Context,
	apiReader client.Reader,
	namespace string,
	ko *svcapitypes.Function,
) error {
	if ko.Spec.KMSKeyRef != nil &&
		ko.Spec.KMSKeyRef.From != nil {
		arr := ko.Spec.KMSKeyRef.From
		if arr == nil || arr.Name == nil || *arr.Name == "" {
			return fmt.Errorf("provided resource reference is nil or empty")
		}
		namespacedName := types.NamespacedName{
			Namespace: namespace,
			Name:      *arr.Name,
		}
		obj := kmsapitypes.Key{}
		err := apiReader.Get(ctx, namespacedName, &obj)
		if err != nil {
			return err
		}
		var refResourceSynced, refResourceTerminal bool
		for _, cond := range obj.Status.Conditions {
			if cond.Type == ackv1alpha1.ConditionTypeResourceSynced &&
				cond.Status == corev1.ConditionTrue {
				refResourceSynced = true
			}
			if cond.Type == ackv1alpha1.ConditionTypeTerminal &&
				cond.Status == corev1.ConditionTrue {
				refResourceTerminal = true
			}
		}
		if refResourceTerminal {
			return ackerr.ResourceReferenceTerminalFor(
				"Key",
				namespace, *arr.Name)
		}
		if !refResourceSynced {
			return ackerr.ResourceReferenceNotSyncedFor(
				"Key",
				namespace, *arr.Name)
		}
		if obj.Status.ACKResourceMetadata == nil || obj.Status.ACKResourceMetadata.ARN == nil {
			return ackerr.ResourceReferenceMissingTargetFieldFor(
				"Key",
				namespace, *arr.Name,
				"Status.ACKResourceMetadata.ARN")
		}
		referencedValue := string(*obj.Status.ACKResourceMetadata.ARN)
		ko.Spec.KMSKeyARN = &referencedValue
	}
	return nil
}

// resolveReferenceForVPCConfig_SecurityGroupIDs reads the resource referenced
// from VPCConfig.SecurityGroupRefs field and sets the VPCConfig.SecurityGroupIDs
// from referenced resource
func resolveReferenceForVPCConfig_SecurityGroupIDs(
	ctx context.Context,
	apiReader client.Reader,
	namespace string,
	ko *svcapitypes.Function,
) error {
	if ko.Spec.VPCConfig == nil {
		return nil
	}

	if ko.Spec.VPCConfig.SecurityGroupRefs != nil &&
		len(ko.Spec.VPCConfig.SecurityGroupRefs) > 0 {
		resolvedReferences := []*string{}
		for _, arrw := range ko.Spec.VPCConfig.SecurityGroupRefs {
			arr := arrw.From
			if arr == nil || arr.Name == nil || *arr.Name == "" {
				return fmt.Errorf("provided resource reference is nil or empty")
			}
			namespacedName := types.NamespacedName{
				Namespace: namespace,
				Name:      *arr.Name,
			}
			obj := ec2apitypes.SecurityGroup{}
			err := apiReader.Get(ctx, namespacedName, &obj)
			if err != nil {
				return err
			}
			var refResourceSynced, refResourceTerminal bool
			for _, cond := range obj.Status.Conditions {
				if cond.Type == ackv1alpha1.ConditionTypeResourceSynced &&
					cond.Status == corev1.ConditionTrue {
					refResourceSynced = true
				}
				if cond.Type == ackv1alpha1.ConditionTypeTerminal &&
					cond.Status == corev1.ConditionTrue {
					refResourceTerminal = true
				}
			}
			if refResourceTerminal {
				return ackerr.ResourceReferenceTerminalFor(
					"SecurityGroup",
					namespace, *arr.Name)
			}
			if !refResourceSynced {
				return ackerr.ResourceReferenceNotSyncedFor(
					"SecurityGroup",
					namespace, *arr.Name)
			}
			if obj.Status.ID == nil {
				return ackerr.ResourceReferenceMissingTargetFieldFor(
					"SecurityGroup",
					namespace, *arr.Name,
					"Status.ID")
			}
			referencedValue := string(*obj.Status.ID)
			resolvedReferences = append(resolvedReferences, &referencedValue)
		}
		ko.Spec.VPCConfig.SecurityGroupIDs = resolvedReferences
	}
	return nil
}

// resolveReferenceForVPCConfig_SubnetIDs reads the resource referenced
// from VPCConfig.SubnetRefs field and sets the VPCConfig.SubnetIDs
// from referenced resource
func resolveReferenceForVPCConfig_SubnetIDs(
	ctx context.Context,
	apiReader client.Reader,
	namespace string,
	ko *svcapitypes.Function,
) error {
	if ko.Spec.VPCConfig == nil {
		return nil
	}

	if ko.Spec.VPCConfig.SubnetRefs != nil &&
		len(ko.Spec.VPCConfig.SubnetRefs) > 0 {
		resolvedReferences := []*string{}
		for _, arrw := range ko.Spec.VPCConfig.SubnetRefs {
			arr := arrw.From
			if arr == nil || arr.Name == nil || *arr.Name == "" {
				return fmt.Errorf("provided resource reference is nil or empty")
			}
			namespacedName := types.NamespacedName{
				Namespace: namespace,
				Name:      *arr.Name,
			}
			obj := ec2apitypes.Subnet{}
			err := apiReader.Get(ctx, namespacedName, &obj)
			if err != nil {
				return err
			}
			var refResourceSynced, refResourceTerminal bool
			for _, cond := range obj.Status.Conditions {
				if cond.Type == ackv1alpha1.ConditionTypeResourceSynced &&
					cond.Status == corev1.ConditionTrue {
					refResourceSynced = true
				}
				if cond.Type == ackv1alpha1.ConditionTypeTerminal &&
					cond.Status == corev1.ConditionTrue {
					refResourceTerminal = true
				}
			}
			if refResourceTerminal {
				return ackerr.ResourceReferenceTerminalFor(
					"Subnet",
					namespace, *arr.Name)
			}
			if !refResourceSynced {
				return ackerr.ResourceReferenceNotSyncedFor(
					"Subnet",
					namespace, *arr.Name)
			}
			if obj.Status.SubnetID == nil {
				return ackerr.ResourceReferenceMissingTargetFieldFor(
					"Subnet",
					namespace, *arr.Name,
					"Status.SubnetID")
			}
			referencedValue := string(*obj.Status.SubnetID)
			resolvedReferences = append(resolvedReferences, &referencedValue)
		}
		ko.Spec.VPCConfig.SubnetIDs = resolvedReferences
	}
	return nil
}
