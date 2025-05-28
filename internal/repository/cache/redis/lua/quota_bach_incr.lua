for i = 1, #KEYS do
    local key = KEYS[i]
    local quota = tonumber(ARGV[i])
    local current = tonumber(redis.call('GET', key) or 0)

    if current < 0 then
        -- 当前配额为负数，直接设置参数
        redis.call('SET', key, quota)
    else
        redis.call('INCRBY', key, quota)
    end
end

return 1
