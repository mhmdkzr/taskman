CREATE TABLE audit_logs (
    request_id text NOT NULL,
    "timestamp" timestamp with time zone NOT NULL,
    duration_ns bigint NOT NULL,
    request_method text NOT NULL,
    request_path text NOT NULL,
    request_query text DEFAULT ''::text NOT NULL,
    request_remote_addr text DEFAULT ''::text NOT NULL,
    request_user_agent text DEFAULT ''::text NOT NULL,
    request_proto text DEFAULT ''::text NOT NULL,
    request_headers jsonb DEFAULT '{}'::jsonb NOT NULL,
    request_body jsonb,
    response_status_code integer NOT NULL,
    response_headers jsonb DEFAULT '{}'::jsonb NOT NULL,
    response_body jsonb,
    response_size bigint DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT audit_logs_duration_ns_check CHECK ((duration_ns >= 0)),
    CONSTRAINT audit_logs_response_size_check CHECK ((response_size >= 0)),
    CONSTRAINT audit_logs_pkey PRIMARY KEY (request_id)
);

CREATE INDEX idx_audit_logs_request_method ON audit_logs USING btree (request_method);
CREATE INDEX idx_audit_logs_request_path ON audit_logs USING btree (request_path);
CREATE INDEX idx_audit_logs_response_status_code ON audit_logs USING btree (response_status_code);
CREATE INDEX idx_audit_logs_timestamp ON audit_logs USING btree ("timestamp" DESC);
