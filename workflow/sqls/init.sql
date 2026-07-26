-- Workflow 框架数据库初始化脚本
-- 数据库: MySQL
-- 建议字符集: utf8mb4

CREATE DATABASE IF NOT EXISTS `landc_workflow`
  DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

USE `landc_workflow`;

-- ============================================================
-- 1. wf_workflows — 工作流定义
-- ============================================================
CREATE TABLE IF NOT EXISTS `wf_workflows` (
  `id`          varchar(64)  NOT NULL COMMENT '工作流ID',
  `name`        varchar(256) NOT NULL COMMENT '工作流名称',
  `description` varchar(1024) DEFAULT NULL COMMENT '描述',
  `version`     int          NOT NULL DEFAULT 1 COMMENT '版本号',
  `status`      varchar(32)  NOT NULL DEFAULT 'ACTIVE' COMMENT '状态: ACTIVE/PAUSED/ARCHIVED',
  `timeout`     bigint       DEFAULT NULL COMMENT '超时时间(秒),0表示不超时',
  `max_retries` int          NOT NULL DEFAULT 0 COMMENT '全局最大重试次数',
  `created_at`  datetime(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
  `updated_at`  datetime(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3) COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `idx_wf_workflows_name` (`name`),
  KEY `idx_wf_workflows_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='工作流定义';

-- ============================================================
-- 2. wf_nodes — DAG 节点
-- ============================================================
CREATE TABLE IF NOT EXISTS `wf_nodes` (
  `id`               varchar(64)  NOT NULL COMMENT '节点ID',
  `workflow_id`      varchar(64)  NOT NULL COMMENT '所属工作流ID',
  `name`             varchar(256) NOT NULL COMMENT '节点名称',
  `type`             varchar(32)  NOT NULL COMMENT '节点类型: HTTP/SCRIPT/DELAY/CONDITION/SWITCH/INPUT/OUTPUT/HUMAN_INPUT/...',
  `description`      varchar(1024) DEFAULT NULL COMMENT '描述',
  `timeout`          bigint       DEFAULT NULL COMMENT '节点超时(秒)',
  `max_retries`      int          NOT NULL DEFAULT 0 COMMENT '重试次数',
  `retry_delay`      bigint       NOT NULL DEFAULT 0 COMMENT '重试间隔(秒)',
  `retry_mode`       varchar(32)  NOT NULL DEFAULT 'LINEAR' COMMENT '重试模式: LINEAR/EXPONENTIAL',
  `retry_max_delay`  bigint       NOT NULL DEFAULT 300 COMMENT '最大重试延迟(秒)',
  `skip_on_failure`  tinyint(1)   NOT NULL DEFAULT 0 COMMENT '失败时跳过继续执行下游',
  `config`           text         DEFAULT NULL COMMENT '节点配置(JSON)',
  `input_mapping`    text         DEFAULT NULL COMMENT '输入映射(JSON)',
  `output_mapping`   text         DEFAULT NULL COMMENT '输出映射(JSON)',
  `condition_expr`   varchar(1024) DEFAULT NULL COMMENT '条件表达式',
  `parallel_branches` int         NOT NULL DEFAULT 1 COMMENT '并行分支数',
  `order_no`         int          NOT NULL DEFAULT 0 COMMENT '排序号',
  `created_at`       datetime(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
  `updated_at`       datetime(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3) COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `idx_wf_nodes_workflow` (`workflow_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='DAG节点';

-- ============================================================
-- 3. wf_edges — DAG 边（支持端口级条件分支）
-- ============================================================
CREATE TABLE IF NOT EXISTS `wf_edges` (
  `id`             varchar(64)  NOT NULL COMMENT '边ID',
  `workflow_id`    varchar(64)  NOT NULL COMMENT '所属工作流ID',
  `source_id`      varchar(64)  NOT NULL COMMENT '上游节点ID',
  `target_id`      varchar(64)  NOT NULL COMMENT '下游节点ID',
  `source_port`    varchar(64)  DEFAULT NULL COMMENT '源端口(条件分支,如 true/false/分支名)',
  `target_port`    varchar(64)  DEFAULT NULL COMMENT '目标端口',
  `condition_expr` varchar(1024) DEFAULT NULL COMMENT '条件表达式',
  `label`          varchar(64)  DEFAULT NULL COMMENT '边标签',
  `internal`       tinyint(1)   NOT NULL DEFAULT 0 COMMENT '内部边(Loop回边,不参与DAG调度)',
  `order_no`       int          NOT NULL DEFAULT 0 COMMENT '排序号',
  `created_at`     datetime(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
  PRIMARY KEY (`id`),
  KEY `idx_wf_edges_workflow` (`workflow_id`),
  KEY `idx_wf_edges_source` (`source_id`),
  KEY `idx_wf_edges_target` (`target_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='DAG边';

-- ============================================================
-- 4. wf_executions — 工作流执行实例
-- ============================================================
CREATE TABLE IF NOT EXISTS `wf_executions` (
  `id`              varchar(64)  NOT NULL COMMENT '执行ID',
  `workflow_id`     varchar(64)  NOT NULL COMMENT '工作流ID',
  `workflow_name`   varchar(256) DEFAULT NULL COMMENT '快照:工作流名称',
  `workflow_ver`    int          DEFAULT NULL COMMENT '快照:工作流版本',
  `trigger_type`    varchar(32)  NOT NULL DEFAULT 'API' COMMENT '触发类型: MANUAL/SCHEDULE/EVENT/API',
  `trigger_id`      varchar(128) DEFAULT NULL COMMENT '外部触发ID(用于幂等)',
  `status`          varchar(32)  NOT NULL DEFAULT 'PENDING' COMMENT '执行状态: PENDING/RUNNING/PAUSED/COMPLETED/FAILED/CANCELLED/TIMEOUT',
  `input`           longtext    DEFAULT NULL COMMENT '输入数据',
  `output`          longtext    DEFAULT NULL COMMENT '输出数据',
  `state_data`      longtext    DEFAULT NULL COMMENT '状态快照(用于暂停恢复)',
  `current_node_id` varchar(64)  DEFAULT NULL COMMENT '当前节点ID(可重入恢复点)',
  `error`           text        DEFAULT NULL COMMENT '错误信息',
  `timeout`         bigint       DEFAULT NULL COMMENT '超时时间(秒)',
  `expires_at`      datetime(3)  DEFAULT NULL COMMENT '超时截止时间',
  `version`         int          NOT NULL DEFAULT 1 COMMENT '乐观锁版本',
  `started_at`      datetime(3)  DEFAULT NULL COMMENT '开始时间',
  `finished_at`     datetime(3)  DEFAULT NULL COMMENT '完成时间',
  `created_at`      datetime(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
  `updated_at`      datetime(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3) COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `idx_wf_executions_workflow` (`workflow_id`),
  KEY `idx_wf_executions_trigger` (`trigger_id`),
  KEY `idx_wf_executions_status` (`status`),
  KEY `idx_wf_executions_expires` (`expires_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='工作流执行实例';

-- ============================================================
-- 5. wf_tasks — 节点执行实例
-- ============================================================
CREATE TABLE IF NOT EXISTS `wf_tasks` (
  `id`           varchar(64)  NOT NULL COMMENT '任务ID',
  `execution_id` varchar(64)  NOT NULL COMMENT '所属执行ID',
  `node_id`      varchar(64)  NOT NULL COMMENT '节点ID',
  `node_name`    varchar(256) DEFAULT NULL COMMENT '快照:节点名称',
  `node_type`    varchar(32)  DEFAULT NULL COMMENT '快照:节点类型',
  `status`       varchar(32)  NOT NULL DEFAULT 'PENDING' COMMENT '任务状态: PENDING/RUNNING/COMPLETED/FAILED/SKIPPED/CANCELLED/RETRYING',
  `input`        longtext    DEFAULT NULL COMMENT '输入数据',
  `output`       longtext    DEFAULT NULL COMMENT '输出数据',
  `error`        text        DEFAULT NULL COMMENT '错误信息',
  `retry_count`  int          NOT NULL DEFAULT 0 COMMENT '已重试次数',
  `max_retries`  int          NOT NULL DEFAULT 0 COMMENT '最大重试次数',
  `is_retry`     tinyint(1)   NOT NULL DEFAULT 0 COMMENT '是否重试执行',
  `attempt_id`   varchar(128) DEFAULT NULL COMMENT '执行尝试ID(用于幂等)',
  `worker_id`    varchar(128) DEFAULT NULL COMMENT '处理该任务的WorkerID',
  `scheduled_at` datetime(3)  DEFAULT NULL COMMENT '计划执行时间',
  `started_at`   datetime(3)  DEFAULT NULL COMMENT '开始时间',
  `finished_at`  datetime(3)  DEFAULT NULL COMMENT '完成时间',
  `created_at`   datetime(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
  `updated_at`   datetime(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3) COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `idx_wf_tasks_execution` (`execution_id`),
  KEY `idx_wf_tasks_node` (`node_id`),
  KEY `idx_wf_tasks_status` (`status`),
  KEY `idx_wf_tasks_attempt` (`attempt_id`),
  KEY `idx_wf_tasks_worker` (`worker_id`),
  KEY `idx_wf_tasks_scheduled` (`scheduled_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='节点执行实例';

-- ============================================================
-- 6. wf_workers — Worker 节点注册
-- ============================================================
CREATE TABLE IF NOT EXISTS `wf_workers` (
  `id`         varchar(128) NOT NULL COMMENT 'WorkerID',
  `address`    varchar(256) NOT NULL COMMENT 'Worker地址',
  `group`      varchar(64)  DEFAULT NULL COMMENT 'Worker分组',
  `status`     varchar(32)  NOT NULL DEFAULT 'ACTIVE' COMMENT '状态',
  `tags`       varchar(512) DEFAULT NULL COMMENT '标签(JSON数组)',
  `heartbeat`  datetime(3)  NOT NULL COMMENT '最近心跳时间',
  `started_at` datetime(3)  NOT NULL COMMENT '启动时间',
  `created_at` datetime(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
  `updated_at` datetime(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3) COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `idx_wf_workers_group` (`group`),
  KEY `idx_wf_workers_heartbeat` (`heartbeat`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Worker节点注册';
