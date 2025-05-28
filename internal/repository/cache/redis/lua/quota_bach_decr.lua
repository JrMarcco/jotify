for i = 1, #KEYS do
    local key = KEYS[i]
    local threshold = tonumber(ARGV[i])
    local current = tonumber(redis.call('GET', key) or 0)

    -- 配额不足，直接返回
    if current < threshold then
        return key
    end
end


for i = 1, #KEYS do
    local key = KEYS[i]
    local quota = tonumber(ARGV[i])
    redis.call('DECRBY', key, quota)
end

return ""
