# code check list
- 함수명은 동사로 시작
- 데이베이스 FK constraint 는 코드에서도 이중 체크
- gorm model.where.scan -> find 으로 축약
- make lint 테스트
- goland inspect code 테스트

# 알림센터
솔루션에서 상태 정보를 수신하고, 사용자에게 솔루션 상태 정보를 송신하는 역할을 한다.

또한, 상태 정보를 기록하고, 사용자 요청이 있을 시, 이 기록된 상태 정보들을 사용자에게 보여준다.

이하에서 상태 정보는 이벤트(Event), 기록된 상태 정보들은 이벤트 히스토리(Event History)라 명칭 한다.

이벤트는 생성, 삭제, 읽기가 가능하며, 수정은 불가하다.

## 이벤트 생성
이벤트 생성은 브로커를 통해 읽은 정보를 통해 생성되며, 사용자의 직접적인 생성은 불가능하다.

이때, 브로커의 입력은 아래와 같아야 한다.
```bash
b, _ := json.Marshal(&model.Event{
		ID:         0,  // 데이터베이스에서 자동 생성하므로, 무시된다.
		TenantID:   0,  // 테넌트가 유효하여야 한다.
		Code:       "", // 이벤트 코드가 유효하여야 한다.
		EventError: "", // 이벤트 에러 입력시 유효하여야 한다.
		Contents:   "", // JSON 형태로 입력을 받으며, 예외적으로 ""(빈 문자열)을 허용한다.
		CreatedAt:  0,  // 이벤트 생성 시간을 뜻하며, 0을 입력 시 알람 센터에서 토픽을 받은 시점으로 설정된다.
	})
_ = broker.Publish(constant.TopicNotifyEvent, &broker.Message{Body: b})
```

## 알림센터 변경으로 유의 사항
- Event 에서 EventCode 의 code 값을 참조
- Event 에 EventError 를 추가하여 Error 발생시 해당 Error 의 code 값을 참조
- 기존의 code 내용을 EventCode 와 EventError 로 분리
  ```
    - 기존 Code: "cdm-center.manager.get_cluster_list.failure-get-bad_request"
    - 변경 후 Event Code: "cdm-center.manager.get_cluster_list.failure-get"
             Event Error Code: "bad_request"
  ```

### createError(context, eventCode, error) 함수 추가 및 사용
  - createError 함수 내부에서 error 를 switch 문으로 분류하여 추가된 error 는 errorCode 와 함께 추가해야함
  ```bash
  func createError(ctx context.Context, eventCode string, err error) error {
    if err == nil {
      return nil
    }

    var errorCode string
    switch {
    case errors.Equal(err, internal.ErrInactiveCluster):
      errorCode = "not_found_cluster"
      return errors.StatusNotFound(ctx, eventCode, errorCode, err)
  
    default:
      // common util 의 CreateError
      if err := util.CreateError(ctx, eventCode, err); err != nil {
          return err
      }
    }
    return nil
  }
  ```
- handler 에서 error return 시 createError 사용. eventCode 에는 errorCode 내용을 제외한 내용까지만 입력.
  단, errors.StatusOK 는 기존방식 그대로 사용.
  ```bash 
    // err
    if err != nil {  
      return createError(ctx, "cdm-center.manager.get_cluster_list.failure-validate_request", err)
    }
  
    // NoContent
    if len(rsp.Clusters) == 0 {
      return createError(ctx, "cdm-center.manager.get_cluster_list.success-get", errors.ErrNoContent)
    }
  
    // success
    return errors.StatusOK(ctx, "cdm-center.manager.get_cluster_list.success", nil)
  ```
  
### ReportEvent(tenantID, eventCode, errorCode, Option) 변경
  - ReportEvent 만 사용시에는 아래와 같이 eventCode 와 errorCode 를 분류하여 사용
  ```
  ex) 
    reportEvent("cdm-dr.manager.main.failure-create_handler", "unusable_broker", err)
  ```
  - 기존에 등록되어 있지 않은 eventCode 나 errorCode 는 dml.sql 에 추가 필요
    - 새로 추가된 eventCode 는 추가 ([event-code-dml.sql](http://github.com/datacommand2/cdm-cloud/documents/-/blob/master/database/event-code-dml.sql))
    ```
    ex) INSERT INTO cdm_event_code (id, code, solution, level, class_1, class_2, class_3) VALUES
      (NEXTVAL ('cdm_event_code_dr_snapshot_seq'), 'cdm-dr.snapshot-manager.delete_all_snapshot.success', 'CDM-DisasterRecovery', 'info', 'Snapshot', '', 'DeleteAllSnapshots');
    ```
    - 새로 추가된 error 는 errorCode 추가. code 가 이미 존재시 추가 불필요 ([event-error-dml.sql](http://github.com/datacommand2/cdm-cloud/documents/-/blob/master/database/event-error-dml.sql))
    ```
    ex) INSERT INTO cdm_event_error (code, solution, service, contents) VALUES
    ('required_parameter', 'CDM-Cloud', 'Common', 'detail: required parameter \n parameter: <%= param %>');
    ```
