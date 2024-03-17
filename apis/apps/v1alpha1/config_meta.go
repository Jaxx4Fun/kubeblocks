/*
Copyright (C) 2022-2023 ApeCloud Co., Ltd

This file is part of KubeBlocks project

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/

package v1alpha1

import "encoding/json"

// MarshalJSON implements the Marshaler interface.
func (c *Payload) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.Data)
}

// UnmarshalJSON implements the Unmarshaler interface.
func (c *Payload) UnmarshalJSON(data []byte) error {
	var out map[string]interface{}
	err := json.Unmarshal(data, &out)
	if err != nil {
		return err
	}
	c.Data = out
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
// This exists here to work around https://github.com/kubernetes/code-generator/issues/50
func (c *Payload) DeepCopyInto(out *Payload) {
	bytes, err := json.Marshal(c.Data)
	if err != nil {
		// TODO how to process error: panic or ignore
		return // ignore
	}
	var clone map[string]interface{}
	err = json.Unmarshal(bytes, &clone)
	if err != nil {
		// TODO how to process error: panic or ignore
		return // ignore
	}
	out.Data = clone
}

func (in *ConfigConstraintSpec) NeedDynamicReloadAction() bool {
	if in.DynamicActionCanBeMerged != nil {
		return !*in.DynamicActionCanBeMerged
	}
	return false
}

func (in *ConfigConstraintSpec) DynamicParametersPolicy() DynamicParameterSelectedPolicy {
	if in.DynamicParameterSelectedPolicy != nil {
		return *in.DynamicParameterSelectedPolicy
	}
	return SelectedDynamicParameters
}

func (configuration *ConfigurationSpec) GetConfigurationItem(name string) *ConfigurationItemDetail {
	for i := range configuration.ConfigItemDetails {
		configItem := &configuration.ConfigItemDetails[i]
		if configItem.Name == name {
			return configItem
		}
	}
	return nil
}

func (configuration *ConfigurationSpec) GetConfigSpec(configSpecName string) *ComponentConfigSpec {
	if configItem := configuration.GetConfigurationItem(configSpecName); configItem != nil {
		return configItem.ConfigSpec
	}
	return nil
}

func (status *ConfigurationStatus) GetItemStatus(name string) *ConfigurationItemDetailStatus {
	for i := range status.ConfigurationItemStatus {
		itemStatus := &status.ConfigurationItemStatus[i]
		if itemStatus.Name == name {
			return itemStatus
		}
	}
	return nil
}