package workspacestorage

import (
	driver "database/sql/driver"
	json "encoding/json"
	fmt "fmt"

	errors "github.com/pkg/errors"

	workspaceservice "github.com/synclet-io/synclet/modules/workspace/workspaceservice"
	// user code 'imports'
	// end user code 'imports'
)

type jsonWorkspaceEventData struct {
	Val         any    `json:"value"`
	OneOfType   string `json:"@type"`
	OneOfTypeID uint   `json:"@type_id"`
}

func (w *jsonWorkspaceEventData) UnmarshalJSON(bytes []byte) error {
	tmp := struct {
		OneOfTypeID uint   `json:"@type_id"`
		OneOfType   string `json:"@type"`
	}{}
	if err := json.Unmarshal(bytes, &tmp); err != nil {
		return fmt.Errorf("unmarshal OneOfType: %w", err)
	}

	switch tmp.OneOfTypeID {
	case 30001:
		var value struct {
			Value jsonWorkspaceCreatedEventData `json:"value"`
		}
		if err := json.Unmarshal(bytes, &value); err != nil {
			return err
		}
		w.Val = &value.Value
	case 30002:
		var value struct {
			Value jsonWorkspaceUpdatedEventData `json:"value"`
		}
		if err := json.Unmarshal(bytes, &value); err != nil {
			return err
		}
		w.Val = &value.Value
	case 30003:
		var value struct {
			Value jsonWorkspaceDeletedEventData `json:"value"`
		}
		if err := json.Unmarshal(bytes, &value); err != nil {
			return err
		}
		w.Val = &value.Value
	}
	return nil
}
func (w *jsonWorkspaceEventData) Scan(value any) error {
	return json.Unmarshal(value.([]byte), w)
}

func (w jsonWorkspaceEventData) Value() (driver.Value, error) {
	return json.Marshal(w)
}

func convertWorkspaceEventDataToDB(val workspaceservice.WorkspaceEventData) (*jsonWorkspaceEventData, error) {
	if val == nil {
		return nil, nil
	}
	result := &jsonWorkspaceEventData{}
	switch v := val.(type) {
	case *workspaceservice.WorkspaceCreatedEventData:
		if v != nil {
			tmp, err := convertWorkspaceCreatedEventDataToJsonModel(v)
			if err != nil {
				return nil, errors.Wrap(err, "convert WorkspaceCreatedEventData to db")
			}
			result.Val = tmp
		} else {
			result.Val = nil
		}
		result.OneOfType = "WorkspaceCreatedEventData"
		result.OneOfTypeID = 30001

		return result, nil
	case *workspaceservice.WorkspaceUpdatedEventData:
		if v != nil {
			tmp, err := convertWorkspaceUpdatedEventDataToJsonModel(v)
			if err != nil {
				return nil, errors.Wrap(err, "convert WorkspaceUpdatedEventData to db")
			}
			result.Val = tmp
		} else {
			result.Val = nil
		}
		result.OneOfType = "WorkspaceUpdatedEventData"
		result.OneOfTypeID = 30002

		return result, nil
	case *workspaceservice.WorkspaceDeletedEventData:
		if v != nil {
			tmp, err := convertWorkspaceDeletedEventDataToJsonModel(v)
			if err != nil {
				return nil, errors.Wrap(err, "convert WorkspaceDeletedEventData to db")
			}
			result.Val = tmp
		} else {
			result.Val = nil
		}
		result.OneOfType = "WorkspaceDeletedEventData"
		result.OneOfTypeID = 30003

		return result, nil
	}
	return nil, fmt.Errorf("invalid WorkspaceEventData value type: %T", val)
}

func convertWorkspaceEventDataFromDB(val *jsonWorkspaceEventData) (workspaceservice.WorkspaceEventData, error) {
	if val == nil {
		return nil, nil
	}

	switch v := (*val).Val.(type) {
	case *jsonWorkspaceCreatedEventData:
		v1, err := convertWorkspaceCreatedEventDataFromJsonModel(v)
		if err != nil {
			return nil, fmt.Errorf("convert WorkspaceCreatedEventData from db: %w", err)
		}

		return v1, nil
	case *jsonWorkspaceUpdatedEventData:
		v1, err := convertWorkspaceUpdatedEventDataFromJsonModel(v)
		if err != nil {
			return nil, fmt.Errorf("convert WorkspaceUpdatedEventData from db: %w", err)
		}

		return v1, nil
	case *jsonWorkspaceDeletedEventData:
		v1, err := convertWorkspaceDeletedEventDataFromJsonModel(v)
		if err != nil {
			return nil, fmt.Errorf("convert WorkspaceDeletedEventData from db: %w", err)
		}

		return v1, nil
	default:
		return nil, fmt.Errorf("invalid WorkspaceEventData value type: %T", *val)
	}

	panic("implement me")
}
