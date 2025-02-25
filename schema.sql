CREATE TABLE `admins` (
  `id` int NOT NULL,
  `user_id` bigint NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE `requests` (
  `id` int NOT NULL,
  `username` varchar(255) DEFAULT NULL,
  `date` datetime DEFAULT NULL,
  `date_done` datetime DEFAULT NULL,
  `user_id` bigint DEFAULT NULL,
  `is_completed` tinyint(1) DEFAULT '0'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE `request_items` (
  `id` int NOT NULL,
  `request_id` int DEFAULT NULL,
  `product_name` varchar(255) DEFAULT NULL,
  `quantity` varchar(100) DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;


ALTER TABLE `admins`
  ADD PRIMARY KEY (`id`),
  ADD UNIQUE KEY `user_id` (`user_id`);

ALTER TABLE `requests`
  ADD PRIMARY KEY (`id`);

ALTER TABLE `request_items`
  ADD PRIMARY KEY (`id`),
  ADD KEY `request_id` (`request_id`);


ALTER TABLE `admins`
  MODIFY `id` int NOT NULL AUTO_INCREMENT;

ALTER TABLE `requests`
  MODIFY `id` int NOT NULL AUTO_INCREMENT;

ALTER TABLE `request_items`
  MODIFY `id` int NOT NULL AUTO_INCREMENT;


ALTER TABLE `request_items`
  ADD CONSTRAINT `request_items_ibfk_1` FOREIGN KEY (`request_id`) REFERENCES `requests` (`id`);