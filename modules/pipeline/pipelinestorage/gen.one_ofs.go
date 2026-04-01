package pipelinestorage

import (
	driver "database/sql/driver"
	json "encoding/json"
	fmt "fmt"

	errors "github.com/pkg/errors"

	pipelineservice "github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	// user code 'imports'
	// end user code 'imports'
)

type jsonConnectorTaskPayload struct {
	Val         any    `json:"value"`
	OneOfType   string `json:"@type"`
	OneOfTypeID uint   `json:"@type_id"`
}

func (c *jsonConnectorTaskPayload) UnmarshalJSON(bytes []byte) error {
	tmp := struct {
		OneOfTypeID uint   `json:"@type_id"`
		OneOfType   string `json:"@type"`
	}{}
	if err := json.Unmarshal(bytes, &tmp); err != nil {
		return fmt.Errorf("unmarshal OneOfType: %w", err)
	}

	switch tmp.OneOfTypeID {
	case 1:
		var value struct {
			Value jsonCheckPayload `json:"value"`
		}
		if err := json.Unmarshal(bytes, &value); err != nil {
			return err
		}
		c.Val = &value.Value
	case 2:
		var value struct {
			Value jsonSpecPayload `json:"value"`
		}
		if err := json.Unmarshal(bytes, &value); err != nil {
			return err
		}
		c.Val = &value.Value
	case 3:
		var value struct {
			Value jsonDiscoverPayload `json:"value"`
		}
		if err := json.Unmarshal(bytes, &value); err != nil {
			return err
		}
		c.Val = &value.Value
	}

	return nil
}
func (c *jsonConnectorTaskPayload) Scan(value any) error {
	return json.Unmarshal(value.([]byte), c)
}

func (c jsonConnectorTaskPayload) Value() (driver.Value, error) {
	return json.Marshal(c)
}

func convertConnectorTaskPayloadToDB(val pipelineservice.ConnectorTaskPayload) (*jsonConnectorTaskPayload, error) {
	if val == nil {
		return nil, nil
	}
	result := &jsonConnectorTaskPayload{}
	switch v := val.(type) {
	case *pipelineservice.CheckPayload:
		if v != nil {
			tmp, err := convertCheckPayloadToJsonModel(v)
			if err != nil {
				return nil, errors.Wrap(err, "convert CheckPayload to db")
			}
			result.Val = tmp
		} else {
			result.Val = nil
		}
		result.OneOfType = "CheckPayload"
		result.OneOfTypeID = 1

		return result, nil
	case *pipelineservice.SpecPayload:
		if v != nil {
			tmp, err := convertSpecPayloadToJsonModel(v)
			if err != nil {
				return nil, errors.Wrap(err, "convert SpecPayload to db")
			}
			result.Val = tmp
		} else {
			result.Val = nil
		}
		result.OneOfType = "SpecPayload"
		result.OneOfTypeID = 2

		return result, nil
	case *pipelineservice.DiscoverPayload:
		if v != nil {
			tmp, err := convertDiscoverPayloadToJsonModel(v)
			if err != nil {
				return nil, errors.Wrap(err, "convert DiscoverPayload to db")
			}
			result.Val = tmp
		} else {
			result.Val = nil
		}
		result.OneOfType = "DiscoverPayload"
		result.OneOfTypeID = 3

		return result, nil
	}

	return nil, fmt.Errorf("invalid ConnectorTaskPayload value type: %T", val)
}

func convertConnectorTaskPayloadFromDB(val *jsonConnectorTaskPayload) (pipelineservice.ConnectorTaskPayload, error) {
	if val == nil {
		return nil, nil
	}

	switch v := (*val).Val.(type) {
	case *jsonCheckPayload:
		v1, err := convertCheckPayloadFromJsonModel(v)
		if err != nil {
			return nil, fmt.Errorf("convert CheckPayload from db: %w", err)
		}

		return v1, nil
	case *jsonSpecPayload:
		v1, err := convertSpecPayloadFromJsonModel(v)
		if err != nil {
			return nil, fmt.Errorf("convert SpecPayload from db: %w", err)
		}

		return v1, nil
	case *jsonDiscoverPayload:
		v1, err := convertDiscoverPayloadFromJsonModel(v)
		if err != nil {
			return nil, fmt.Errorf("convert DiscoverPayload from db: %w", err)
		}

		return v1, nil
	default:
		return nil, fmt.Errorf("invalid ConnectorTaskPayload value type: %T", *val)
	}

	panic("implement me")
}

type jsonConnectorTaskResult struct {
	Val         any    `json:"value"`
	OneOfType   string `json:"@type"`
	OneOfTypeID uint   `json:"@type_id"`
}

func (c *jsonConnectorTaskResult) UnmarshalJSON(bytes []byte) error {
	tmp := struct {
		OneOfTypeID uint   `json:"@type_id"`
		OneOfType   string `json:"@type"`
	}{}
	if err := json.Unmarshal(bytes, &tmp); err != nil {
		return fmt.Errorf("unmarshal OneOfType: %w", err)
	}

	switch tmp.OneOfTypeID {
	case 4:
		var value struct {
			Value jsonCheckResult `json:"value"`
		}
		if err := json.Unmarshal(bytes, &value); err != nil {
			return err
		}
		c.Val = &value.Value
	case 5:
		var value struct {
			Value jsonSpecResult `json:"value"`
		}
		if err := json.Unmarshal(bytes, &value); err != nil {
			return err
		}
		c.Val = &value.Value
	case 6:
		var value struct {
			Value jsonDiscoverResult `json:"value"`
		}
		if err := json.Unmarshal(bytes, &value); err != nil {
			return err
		}
		c.Val = &value.Value
	}

	return nil
}
func (c *jsonConnectorTaskResult) Scan(value any) error {
	return json.Unmarshal(value.([]byte), c)
}

func (c jsonConnectorTaskResult) Value() (driver.Value, error) {
	return json.Marshal(c)
}

func convertConnectorTaskResultToDB(val pipelineservice.ConnectorTaskResult) (*jsonConnectorTaskResult, error) {
	if val == nil {
		return nil, nil
	}
	result := &jsonConnectorTaskResult{}
	switch v := val.(type) {
	case *pipelineservice.CheckResult:
		if v != nil {
			tmp, err := convertCheckResultToJsonModel(v)
			if err != nil {
				return nil, errors.Wrap(err, "convert CheckResult to db")
			}
			result.Val = tmp
		} else {
			result.Val = nil
		}
		result.OneOfType = "CheckResult"
		result.OneOfTypeID = 4

		return result, nil
	case *pipelineservice.SpecResult:
		if v != nil {
			tmp, err := convertSpecResultToJsonModel(v)
			if err != nil {
				return nil, errors.Wrap(err, "convert SpecResult to db")
			}
			result.Val = tmp
		} else {
			result.Val = nil
		}
		result.OneOfType = "SpecResult"
		result.OneOfTypeID = 5

		return result, nil
	case *pipelineservice.DiscoverResult:
		if v != nil {
			tmp, err := convertDiscoverResultToJsonModel(v)
			if err != nil {
				return nil, errors.Wrap(err, "convert DiscoverResult to db")
			}
			result.Val = tmp
		} else {
			result.Val = nil
		}
		result.OneOfType = "DiscoverResult"
		result.OneOfTypeID = 6

		return result, nil
	}

	return nil, fmt.Errorf("invalid ConnectorTaskResult value type: %T", val)
}

func convertConnectorTaskResultFromDB(val *jsonConnectorTaskResult) (pipelineservice.ConnectorTaskResult, error) {
	if val == nil {
		return nil, nil
	}

	switch v := (*val).Val.(type) {
	case *jsonCheckResult:
		v1, err := convertCheckResultFromJsonModel(v)
		if err != nil {
			return nil, fmt.Errorf("convert CheckResult from db: %w", err)
		}

		return v1, nil
	case *jsonSpecResult:
		v1, err := convertSpecResultFromJsonModel(v)
		if err != nil {
			return nil, fmt.Errorf("convert SpecResult from db: %w", err)
		}

		return v1, nil
	case *jsonDiscoverResult:
		v1, err := convertDiscoverResultFromJsonModel(v)
		if err != nil {
			return nil, fmt.Errorf("convert DiscoverResult from db: %w", err)
		}

		return v1, nil
	default:
		return nil, fmt.Errorf("invalid ConnectorTaskResult value type: %T", *val)
	}

	panic("implement me")
}
