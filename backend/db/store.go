package db

import (
	"context"

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
					language,
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
					:language,
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
			language = :language,
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
			title = :title
		WHERE saved_title_id = :saved_title_id
			AND number = :number
			AND language IS :language
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
			language,
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
			:language,
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
