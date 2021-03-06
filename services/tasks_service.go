package services

import (
	"time"
	"net/url"
	"github.com/wilsontamarozzi/panda-api/services/models"
	"github.com/wilsontamarozzi/panda-api/helpers"
	"github.com/wilsontamarozzi/panda-api/logger"
)

func GetTasks(pag helpers.Pagination, q url.Values, userRequest string) models.Tasks {

	var tasks models.Tasks
	var typeDateQuery string

	db := Con

	if q.Get("title") != "" {
		db = db.Where("title iLIKE ?", "%" + q.Get("title") + "%")	
	}

	if q.Get("situation") == "open" {
		db = db.Where("completed_at IS NULL")
	}

	if q.Get("situation") == "done" {
		db = db.Where("completed_at IS NOT NULL")
	}

	switch q.Get("assigned") {
		case "author":	db = db.Where("registered_by_uuid = ?", userRequest)
		case "all": 	// all task
		default: 		db = db.Where("assignee_uuid = ?", userRequest)
	}

	switch q.Get("type_date") {
		case "registered":	typeDateQuery = "registered_at"
		case "due": 		typeDateQuery = "due"
		case "completed": 	typeDateQuery = "completed_at"
		default: 			typeDateQuery = "registered_at"
	}

	startDate := q.Get("start_date")
	endDate := q.Get("end_date")

	if startDate != "" && endDate != "" {
		db = db.Where(typeDateQuery + "::DATE BETWEEN ? AND ?", startDate, endDate)
	} else if startDate != "" {
		db = db.Where(typeDateQuery + "::DATE >= ?", startDate)
	} else if endDate != "" {
		db = db.Where(typeDateQuery + "::DATE <= ?", endDate)
	}
	
	db.Preload("Category").
		Preload("RegisteredBy").
		Preload("Assignee").
		Preload("Person").
		Limit(pag.ItemPerPage).
		Offset(pag.StartIndex).
		Order("registered_at desc").
		Find(&tasks)

    return tasks
}

func GetTask(taskId string) models.Task {

	var task models.Task

	Con.Preload("Category").
		Preload("RegisteredBy").
		Preload("Assignee").
		Preload("TaskHistorics.RegisteredBy").
		Preload("Person").
		Where("uuid = ?", taskId).
		First(&task)

	return task
}

func DeleteTask(taskId string) error {
	err := Con.Where("uuid = ?", taskId).Delete(&models.Task{}).Error

	if err != nil {
		logger.Fatal(err)
	}

	return err;
}

func CreateTask(task models.Task) (models.Task, error) {
	
	record := models.Task{
		Title 				: task.Title,
		Due 				: task.Due,
		Visualized 			: false,
		CompletedAt 		: task.CompletedAt,
		RegisteredAt 		: time.Now(),
		RegisteredByUUID 	: task.RegisteredByUUID,
		CategoryUUID 		: task.Category.UUID,
		PersonUUID 			: task.Person.UUID,
		AssigneeUUID 		: task.Assignee.UUID,
	}

	err := Con.Set("gorm:save_associations", false).
		Create(&record).Error

	if err != nil {
		logger.Fatal(err)
	} else {
		historics, err := CreateTaskComment(task.TaskHistorics, task.RegisteredByUUID, record.UUID)

		if err != nil {
			logger.Fatal(err)
		} else {
			record.TaskHistorics = historics
		}
	}

	return record, err
}

func UpdateTask(task models.Task) (models.Task, error) {
	
	record := models.Task{
		Title 			: task.Title,
		Due 			: task.Due,
		CompletedAt 	: task.CompletedAt,
		CategoryUUID 	: task.Category.UUID,
		PersonUUID 		: task.Person.UUID,
		AssigneeUUID 	: task.Assignee.UUID,
	}

	err := Con.Set("gorm:save_associations", false).
		Model(&models.Task{}).
		Where("uuid = ?", task.UUID).
		Updates(&record).Error

	if err != nil {
		logger.Fatal(err)
	} else {
		historics, err := CreateTaskComment(task.TaskHistorics, task.RegisteredByUUID, task.UUID)

		if err != nil {
			logger.Fatal(err)
		} else {
			record.TaskHistorics = historics
		}
	}

	return record, err
}

func CountRowsTask() int {
	var count int
	Con.Model(&models.Task{}).Count(&count)

	return count
}

func CreateTaskComment(historics models.TaskHistorics, registeredByUUID string, taskUUID string) (models.TaskHistorics, error) {
	
	for _, historic := range historics {
		historic.RegisteredByUUID 	= registeredByUUID
		historic.RegisteredAt 		= time.Now()
		historic.TaskUUID 			= taskUUID

		if err := Con.Set("gorm:save_associations", false).Create(&historic).Error; err != nil {
			return historics, err
		}
	}

	return historics, nil
}