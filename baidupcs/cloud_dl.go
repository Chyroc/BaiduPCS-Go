package baidupcs

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/iikira/BaiduPCS-Go/pcstable"
	"github.com/iikira/BaiduPCS-Go/pcsutil/converter"
	"github.com/iikira/BaiduPCS-Go/pcsutil/pcstime"
	"github.com/json-iterator/go"
)

// CloudDlFileInfo 离线下载的文件信息
type CloudDlFileInfo struct {
	FileName string `json:"file_name"`
	FileSize int64  `json:"file_size"`
}

// CloudDlTaskInfo 离线下载的任务信息
type CloudDlTaskInfo struct {
	TaskID       int64
	Status       int // 0下载成功, 1下载进行中, 2系统错误, 3资源不存在, 4下载超时, 5资源存在但下载失败, 6存储空间不足, 7任务取消
	StatusText   string
	FileSize     int64  // 文件大小
	FinishedSize int64  // 文件大小
	CreateTime   int64  // 创建时间
	StartTime    int64  // 开始时间
	FinishTime   int64  // 结束时间
	SavePath     string // 保存的路径
	SourceURL    string // 资源地址
	TaskName     string // 任务名称, 一般为文件名
	OdType       int
	FileList     []*CloudDlFileInfo
	Result       int // 0查询成功，结果有效，1要查询的task_id不存在
}

// CloudDlTaskList 离线下载的任务信息列表
type CloudDlTaskList []*CloudDlTaskInfo

// cloudDlTaskInfo 用于解析远程返回的JSON
type cloudDlTaskInfo struct {
	Status       string `json:"status"`
	FileSize     string `json:"file_size"`
	FinishedSize string `json:"finished_size"`
	CreateTime   string `json:"create_time"`
	StartTime    string `json:"start_time"`
	FinishTime   string `json:"finish_time"`
	SavePath     string `json:"save_path"`
	SourceURL    string `json:"source_url"`
	TaskName     string `json:"task_name"`
	OdType       string `json:"od_type"`
	FileList     []*struct {
		FileName string `json:"file_name"`
		FileSize string `json:"file_size"`
	} `json:"file_list"`
	Result int `json:"result"`
}

func (ci *cloudDlTaskInfo) convert() *CloudDlTaskInfo {
	ci2 := &CloudDlTaskInfo{
		Status:       converter.MustInt(ci.Status),
		FileSize:     converter.MustInt64(ci.FileSize),
		FinishedSize: converter.MustInt64(ci.FinishedSize),
		CreateTime:   converter.MustInt64(ci.CreateTime),
		StartTime:    converter.MustInt64(ci.StartTime),
		FinishTime:   converter.MustInt64(ci.FinishTime),
		SavePath:     ci.SavePath,
		SourceURL:    ci.SourceURL,
		TaskName:     ci.TaskName,
		OdType:       converter.MustInt(ci.OdType),
		Result:       ci.Result,
	}

	ci2.FileList = make([]*CloudDlFileInfo, 0, len(ci.FileList))
	for _, v := range ci.FileList {
		if v == nil {
			continue
		}

		ci2.FileList = append(ci2.FileList, &CloudDlFileInfo{
			FileName: v.FileName,
			FileSize: converter.MustInt64(v.FileSize),
		})
	}

	return ci2
}

// CloudDlAddTask 添加离线下载任务
func (pcs *BaiduPCS) CloudDlAddTask(sourceURL, savePath string) (taskID int64, pcsError Error) {
	dataReadCloser, pcsError := pcs.PrepareCloudDlAddTask(sourceURL, savePath)
	if pcsError != nil {
		return
	}

	defer dataReadCloser.Close()

	errInfo := NewErrorInfo(OperationCloudDlAddTask)
	taskInfo := &struct {
		TaskID int64 `json:"task_id"`
		*ErrInfo
	}{
		ErrInfo: errInfo,
	}

	d := jsoniter.NewDecoder(dataReadCloser)
	err := d.Decode(taskInfo)
	if err != nil {
		errInfo.jsonError(err)
		return 0, errInfo
	}

	if taskInfo.ErrCode != 0 {
		return 0, taskInfo.ErrInfo
	}

	return taskInfo.TaskID, nil
}

func (pcs *BaiduPCS) cloudDlQueryTask(op string, taskIDs []int64) (cl CloudDlTaskList, pcsError Error) {
	errInfo := NewErrorInfo(op)
	if len(taskIDs) == 0 {
		errInfo.errType = ErrTypeOthers
		errInfo.err = fmt.Errorf("no input any task_ids")
		return nil, errInfo
	}

	taskStrIDs := make([]string, len(taskIDs))
	for k := range taskStrIDs {
		taskStrIDs[k] = strconv.FormatInt(taskIDs[k], 10)
	}

	dataReadCloser, pcsError := pcs.PrepareCloudDlQueryTask(strings.Join(taskStrIDs, ","))
	if pcsError != nil {
		return
	}

	defer dataReadCloser.Close()

	taskInfo := &struct {
		TaskInfo map[string]*cloudDlTaskInfo `json:"task_info"`
		*ErrInfo
	}{
		ErrInfo: errInfo,
	}

	d := jsoniter.NewDecoder(dataReadCloser)
	err := d.Decode(taskInfo)
	if err != nil {
		errInfo.jsonError(err)
		return nil, errInfo
	}

	if taskInfo.ErrCode != 0 {
		return nil, taskInfo.ErrInfo
	}

	var v2 *CloudDlTaskInfo
	cl = make(CloudDlTaskList, 0, len(taskStrIDs))
	for k := range taskStrIDs {
		v := taskInfo.TaskInfo[taskStrIDs[k]]
		if v == nil {
			continue
		}

		v2 = v.convert()

		v2.TaskID, err = strconv.ParseInt(taskStrIDs[k], 10, 64)
		if err != nil {
			continue
		}

		v2.ParseText()
		cl = append(cl, v2)
	}

	return cl, nil
}

// CloudDlQueryTask 精确查询离线下载任务
func (pcs *BaiduPCS) CloudDlQueryTask(taskIDs []int64) (cl CloudDlTaskList, pcsError Error) {
	return pcs.cloudDlQueryTask(OperationCloudDlQueryTask, taskIDs)
}

// CloudDlListTask 查询离线下载任务列表
func (pcs *BaiduPCS) CloudDlListTask() (cl CloudDlTaskList, pcsError Error) {
	dataReadCloser, pcsError := pcs.PrepareCloudDlListTask()
	if pcsError != nil {
		return
	}

	defer dataReadCloser.Close()

	errInfo := NewErrorInfo(OperationCloudDlListTask)
	taskInfo := &struct {
		TaskInfo []*struct {
			TaskID string `json:"task_id"`
		} `json:"task_info"`
		*ErrInfo
	}{
		ErrInfo: errInfo,
	}

	d := jsoniter.NewDecoder(dataReadCloser)
	err := d.Decode(taskInfo)
	if err != nil {
		errInfo.jsonError(err)
		return nil, errInfo
	}

	if taskInfo.ErrCode != 0 {
		return nil, taskInfo.ErrInfo
	}

	// 没有任务
	if len(taskInfo.TaskInfo) <= 0 {
		return CloudDlTaskList{}, nil
	}

	var (
		taskID  int64
		taskIDs = make([]int64, 0, len(taskInfo.TaskInfo))
	)
	for _, v := range taskInfo.TaskInfo {
		if v == nil {
			continue
		}

		if taskID, err = strconv.ParseInt(v.TaskID, 10, 64); err == nil {
			taskIDs = append(taskIDs, taskID)
		}
	}

	return pcs.cloudDlQueryTask(OperationCloudDlListTask, taskIDs)
}

func (pcs *BaiduPCS) cloudDlManipTask(op string, taskID int64) (pcsError Error) {
	var dataReadCloser io.ReadCloser

	switch op {
	case OperationCloudDlCancelTask:
		dataReadCloser, pcsError = pcs.PrepareCloudDlCancelTask(taskID)
	case OperationCloudDlDeleteTask:
		dataReadCloser, pcsError = pcs.PrepareCloudDlDeleteTask(taskID)
	default:
		panic("unknown op, " + op)
	}
	if pcsError != nil {
		return
	}

	defer dataReadCloser.Close()

	errInfo := decodeJSONError(op, dataReadCloser)
	return errInfo
}

// CloudDlCancelTask 取消离线下载任务
func (pcs *BaiduPCS) CloudDlCancelTask(taskID int64) (pcsError Error) {
	return pcs.cloudDlManipTask(OperationCloudDlCancelTask, taskID)
}

// CloudDlDeleteTask 删除离线下载任务
func (pcs *BaiduPCS) CloudDlDeleteTask(taskID int64) (pcsError Error) {
	return pcs.cloudDlManipTask(OperationCloudDlDeleteTask, taskID)
}

// ParseText 解析状态码
func (ci *CloudDlTaskInfo) ParseText() {
	switch ci.Status {
	case 0:
		ci.StatusText = "下载成功"
	case 1:
		ci.StatusText = "下载进行中"
	case 2:
		ci.StatusText = "系统错误"
	case 3:
		ci.StatusText = "资源不存在"
	case 4:
		ci.StatusText = "下载超时"
	case 5:
		ci.StatusText = "资源存在但下载失败"
	case 6:
		ci.StatusText = "存储空间不足"
	case 7:
		ci.StatusText = "任务取消"
	default:
		ci.StatusText = "未知状态码: " + strconv.Itoa(ci.Status)
	}
}

func (cl CloudDlTaskList) String() string {
	builder := &strings.Builder{}
	tb := pcstable.NewTable(builder)
	tb.SetHeader([]string{"#", "任务ID", "任务名称", "文件大小", "创建日期", "保存路径", "资源地址", "状态"})
	for k, v := range cl {
		tb.Append([]string{strconv.Itoa(k), strconv.FormatInt(v.TaskID, 10), v.TaskName, converter.ConvertFileSize(v.FileSize), pcstime.FormatTime(v.CreateTime), v.SavePath, v.SourceURL, v.StatusText})
	}
	tb.Render()
	return builder.String()
}
