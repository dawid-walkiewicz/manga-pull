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

	result, err := s.db.NamedExecContext(ctx, `
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
