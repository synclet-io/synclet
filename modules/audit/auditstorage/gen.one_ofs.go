package auditstorage

import (
	driver "database/sql/driver"
	json "encoding/json"
	fmt "fmt"

	errors "github.com/pkg/errors"

	auditservice "github.com/synclet-io/synclet/modules/audit/auditservice"
	// user code 'imports'
	// end user code 'imports'
)

type jsonAuditOutboxEventData struct {
	Val         any    `json:"value"`
	OneOfType   string `json:"@type"`
	OneOfTypeID uint   `json:"@type_id"`
}

func (a *jsonAuditOutboxEventData) UnmarshalJSON(bytes []byte) error {
	tmp := struct {
		OneOfTypeID uint   `json:"@type_id"`
		OneOfType   string `json:"@type"`
	}{}
	if err := json.Unmarshal(bytes, &tmp); err != nil {
		return fmt.Errorf("unmarshal OneOfType: %w", err)
	}

	switch tmp.OneOfTypeID {
	case 50001:
		var value struct {
			Value jsonAuditEventRecordedData `json:"value"`
		}
		if err := json.Unmarshal(bytes, &value); err != nil {
			return err
		}
		a.Val = &value.Value
	}
	return nil
}
func (a *jsonAuditOutboxEventData) Scan(value any) error {
	return json.Unmarshal(value.([]byte), a)
}

func (a jsonAuditOutboxEventData) Value() (driver.Value, error) {
	return json.Marshal(a)
}

func convertAuditOutboxEventDataToDB(val auditservice.AuditOutboxEventData) (*jsonAuditOutboxEventData, error) {
	if val == nil {
		return nil, nil
	}
	result := &jsonAuditOutboxEventData{}
	switch v := val.(type) {
	case *auditservice.AuditEventRecordedData:
		if v != nil {
			tmp, err := convertAuditEventRecordedDataToJsonModel(v)
			if err != nil {
				return nil, errors.Wrap(err, "convert AuditEventRecordedData to db")
			}
			result.Val = tmp
		} else {
			result.Val = nil
		}
		result.OneOfType = "AuditEventRecordedData"
		result.OneOfTypeID = 50001

		return result, nil
	}
	return nil, fmt.Errorf("invalid AuditOutboxEventData value type: %T", val)
}

func convertAuditOutboxEventDataFromDB(val *jsonAuditOutboxEventData) (auditservice.AuditOutboxEventData, error) {
	if val == nil {
		return nil, nil
	}

	switch v := (*val).Val.(type) {
	case *jsonAuditEventRecordedData:
		v1, err := convertAuditEventRecordedDataFromJsonModel(v)
		if err != nil {
			return nil, fmt.Errorf("convert AuditEventRecordedData from db: %w", err)
		}

		return v1, nil
	default:
		return nil, fmt.Errorf("invalid AuditOutboxEventData value type: %T", *val)
	}

	panic("implement me")
}
