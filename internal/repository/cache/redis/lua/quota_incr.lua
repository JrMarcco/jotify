local key = KEYS[1]
local quota = tonumber(ARGV[1])
-- 防止 key 不存在
local current = tonumber(redis.call('GET', key) or 0)

if current < 0 then
    -- 当前配额为负，直接设置配额
    redis.call('SET', key, quota)
    return quota
elseif current > 0 then
    return redis.call('INCRBY', key, quota)
else
    -- key 不存在，直接设置配额
    redis.call('SET', key, quota)
    return quota
end
