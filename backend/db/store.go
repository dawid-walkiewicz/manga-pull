package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

type Store struct {
	db *sqlx.DB
}

func NewStore(db *sqlx.DB) *Store {
	return &Store{db: db}
}

func (s *Store) GetSavedTitle(ctx context.Context, id int64) (SavedTitle, error) {
	var title SavedTitle

	err := s.db.GetContext(ctx, &title,
		`
		SELECT *
		FROM saved_titles
		WHERE id = ?
		`,
		id,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return SavedTitle{}, ErrNotFound
		}

		return SavedTitle{}, err
	}

	return title, nil
}

func (s *Store) FindSavedTitle(ctx context.Context, pluginID string, remoteID string) (SavedTitle, error) {
	var title SavedTitle

	err := s.db.GetContext(ctx, &title,
		`
		SELECT *
		FROM saved_titles
		WHERE plugin_id = ? AND remote_id = ?
		`,
		pluginID, remoteID,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return SavedTitle{}, ErrNotFound
		}

		return SavedTitle{}, err
	}

	return title, nil
}

func (s *Store) ListSavedTitles(ctx context.Context) ([]SavedTitle, error) {
	titles := make([]SavedTitle, 0)

	err := s.db.SelectContext(ctx, &titles,
		`
		SELECT *
		FROM saved_titles
		`,
	)

	if err != nil {
		return nil, err
	}

	return titles, nil
}

func (s *Store) SaveTitle(ctx context.Context, title SavedTitle, chapters []Chapter) (int64, error) {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	result, err := tx.NamedExecContext(ctx, `
		INSERT INTO saved_titles (
			plugin_id,
			remote_id,
			title,
			alternative_titles,
			author,
			artist,
			status,
			description,
			cover,
			url,
			group_filter,
			directory_name,
			chapter_name_template,
			last_refreshed_at
		)
		VALUES (
			:plugin_id,
			:remote_id,
			:title,
			:alternative_titles,
			:author,
			:artist,
			:status,
			:description,
			:cover,
			:url,
			:group_filter,
			:directory_name,
			:chapter_name_template,
			:last_refreshed_at
		)
	`, title)

	if err != nil {
		return 0, err
	}

	titleID, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	for i := range chapters {
		chapters[i].SavedTitleID = titleID

		if _, err := tx.NamedExecContext(ctx, `
				INSERT INTO chapters (
					saved_title_id,
					remote_id,
					number,
					volume,
					season,
					title,
					group_name,
					url,
					published_at,
					downloaded
				)
				VALUES (
					:saved_title_id,
					:remote_id,
					:number,
					:volume,
					:season,
					:title,
					:group_name,
					:url,
					:published_at,
					:downloaded
				)
			`, chapters[i]); err != nil {
			return 0, err
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return titleID, nil
}

func (s *Store) DeleteSavedTitle(ctx context.Context, id int64) error {
	result, err := s.db.ExecContext(
		ctx,
		`DELETE FROM saved_titles WHERE id = ?`,
		id,
	)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

func (s *Store) ListChapters(ctx context.Context, savedTitleID int64) ([]Chapter, error) {
	chapters := make([]Chapter, 0)

	err := s.db.SelectContext(ctx, &chapters,
		`
		SELECT *
		FROM chapters
		WHERE saved_title_id = ?
		ORDER BY published_at DESC;
		`, savedTitleID,
	)

	if err != nil {
		return nil, err
	}

	return chapters, nil
}

func (s *Store) RefreshTitle(ctx context.Context, title SavedTitle, incoming []Chapter) ([]Chapter, error) {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	_, err = tx.NamedExecContext(ctx, `
		UPDATE saved_titles
		SET
		    title = :title,
		    alternative_titles = :alternative_titles,
		    author = :author,
			artist = :artist,
			status = :status,
			description = :description,
			cover = :cover,
			url = :url,
			last_refreshed_at = :last_refreshed_at
		WHERE id = :id;
	`, title)

	if err != nil {
		return nil, err
	}

	updateByRemoteID, err := tx.PrepareNamedContext(ctx, `
		UPDATE chapters
		SET
			number = :number,
			volume = :volume,
			season = :season,
			title = :title,
			group_name = :group_name,
			url = :url,
			published_at = :published_at
		WHERE saved_title_id = :saved_title_id
			AND remote_id = :remote_id
	`)
	if err != nil {
		return nil, err
	}
	defer updateByRemoteID.Close()

	updateByIdentity, err := tx.PrepareNamedContext(ctx, `
		UPDATE chapters
		SET
			remote_id = :remote_id,
			title = :title,
			url = :url
		WHERE saved_title_id = :saved_title_id
			AND number = :number
			AND group_name IS :group_name
				AND (
			        (:volume IS NOT NULL AND volume = :volume)
			     OR (:volume IS NULL AND :season IS NOT NULL AND season = :season)
			     OR (:volume IS NULL AND :season IS NULL AND volume IS NULL AND season IS NULL)
					)
	`)
	if err != nil {
		return nil, err
	}
	defer updateByIdentity.Close()

	insert, err := tx.PrepareNamedContext(ctx, `
		INSERT INTO chapters (
			saved_title_id,
			remote_id,
			number,
			volume,
			season,
			title,
			group_name,
			url,
			published_at,
			downloaded
		)
		VALUES (
			:saved_title_id,
			:remote_id,
			:number,
			:volume,
			:season,
			:title,
			:group_name,
			:url,
			:published_at,
			:downloaded
		)
	`)
	if err != nil {
		return nil, err
	}
	defer insert.Close()

	new := make([]Chapter, 0)
	for i := range incoming {
		result, err := updateByRemoteID.ExecContext(ctx, incoming[i])
		if err != nil {
			return nil, err
		}

		rows, err := result.RowsAffected()
		if err != nil {
			return nil, err
		}

		if rows > 0 {
			continue
		}

		result, err = updateByIdentity.ExecContext(ctx, incoming[i])
		if err != nil {
			return nil, err
		}

		rows, err = result.RowsAffected()
		if err != nil {
			return nil, err
		}

		if rows > 0 {
			continue
		}

		incoming[i].SavedTitleID = title.ID

		if result, err = insert.ExecContext(ctx, incoming[i]); err != nil {
			return nil, err
		}
		id, err := result.LastInsertId()
		if err != nil {
			return nil, err
		}

		incoming[i].ID = id
		new = append(new, incoming[i])
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return new, nil
}

func (s *Store) ListPlugins(ctx context.Context) ([]PluginRecord, error) {
	plugins := make([]PluginRecord, 0)

	err := s.db.SelectContext(ctx, &plugins,
		`
		SELECT *
		FROM plugins
		`,
	)

	if err != nil {
		return nil, err
	}

	return plugins, nil
}

func (s *Store) CreatePlugins(ctx context.Context, plugins []PluginRecord) error {
	if len(plugins) == 0 {
		return nil
	}

	query := `
		INSERT INTO plugins (
			id,
			name,
			enabled,
			path
		)
		VALUES (
			:id,
			:name,
			:enabled,
			:path
		)
		ON CONFLICT (id)
		DO UPDATE SET
    		name = excluded.name,
         	path = excluded.path;
      `

	_, err := s.db.NamedExecContext(ctx, query, plugins)

	if err != nil {
		return err
	}

	return nil
}

func (s *Store) SetPluginEnabled(ctx context.Context, id string, enabled bool) error {
	query := `
		UPDATE plugins
		SET enabled = ?
    	WHERE id = ?
      `

	_, err := s.db.ExecContext(ctx, query, enabled, id)

	if err != nil {
		return err
	}

	return nil
}

func (s *Store) GetJob(ctx context.Context, id int64) (Job, error) {
	var job Job

	err := s.db.GetContext(ctx, &job,
		`
		SELECT *
		FROM jobs
		WHERE id = ?
		`,
		id,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Job{}, ErrNotFound
		}

		return Job{}, err
	}

	return job, nil
}

func (s *Store) ListJobs(ctx context.Context) ([]Job, error) {
	jobs := make([]Job, 0)

	err := s.db.SelectContext(ctx, &jobs,
		`
		SELECT *
		FROM jobs
		`,
	)

	if err != nil {
		return nil, err
	}

	return jobs, nil
}

func (s *Store) CreateJob(ctx context.Context, job Job) (int64, error) {
	query := `
		INSERT INTO jobs (
			job_type,
			status,
			saved_title_id,
			chapter_id,
			created_at
		)
		VALUES (
			:job_type,
			:status,
			:saved_title_id,
			:chapter_id,
			:created_at
		)
      `

	result, err := s.db.NamedExecContext(ctx, query, job)
	if err != nil {
		return 0, err
	}

	jobID, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return jobID, nil
}

func (s *Store) ClaimNextJob(ctx context.Context) (*Job, error) {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var job Job
	err = tx.GetContext(ctx, &job,
		`
		SELECT *
		FROM jobs
		WHERE status = 'queued'
		ORDER BY created_at ASC, id ASC
		LIMIT 1
		`,
	)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	result, err := tx.ExecContext(ctx, `
		UPDATE jobs
		SET
			status = 'running',
			progress = '0',
			attempt = '1',
			started_at = ?
    	WHERE id = ?
     		AND status = 'queued'
      `, now, job.ID)
	if err != nil {
		return nil, err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}

	if rows != 1 {
		return nil, fmt.Errorf("job %d could not be claimed", job.ID)
	}

	err = tx.GetContext(ctx, &job,
		`
		SELECT *
		FROM jobs
		WHERE id = ?
		`, job.ID,
	)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &job, nil
}

func (s *Store) UpdateJobStatus(ctx context.Context, jobStatus JobStatusUpdate) (*Job, error) {
	var query string

	switch jobStatus.Status {
	case "running":
		query = `
				UPDATE jobs
				SET
					status = :status,
					progress = :progress,
					started_at = :started_at
				WHERE id = :id
			`
	case "paused":
		query = `
				UPDATE jobs
				SET
					status = :status,
					progress = :progress
				WHERE id = :id
			`
	case "retrying":
		query = `
				UPDATE jobs
				SET
					status = :status,
					attempt = :attempt
				WHERE id = :id
			`
	case "completed":
		query = `
				UPDATE jobs
				SET
					status = :status,
					progress = :progress,
					error_message = :error_message,
					finished_at = :finished_at
				WHERE id = :id
					AND status IN ('running', 'retrying')
			`
	case "failed":
		query = `
				UPDATE jobs
				SET
					status = :status,
					error_message = :error_message
				WHERE id = :id
					AND status IN ('running', 'retrying')
			`
	case "cancelled":
		query = `
				UPDATE jobs
				SET
					status = :status
				WHERE id = :id
					AND status IN ('queued', 'running', 'retrying', 'paused')
			`
	default:
		return nil, fmt.Errorf("unsupported job status: %q", jobStatus.Status)
	}

	result, err := s.db.NamedExecContext(ctx, query, jobStatus)
	if err != nil {
		return nil, err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if rows != 1 {
		return nil, ErrNotFound
	}

	var updated Job
	err = s.db.GetContext(ctx, &updated,
		`
		SELECT *
		FROM jobs
		WHERE id = ?
		`, jobStatus.ID,
	)

	if err != nil {
		return nil, err
	}

	return &updated, nil
}
