module Utils
  module ConcurrentHTTP
    def self.run_parallel(max_concurrent : Int32, tasks : Array(Proc(T))) : Array(T) forall T
      return [] of T if tasks.empty?

      semaphore = Channel(Nil).new(max_concurrent)
      channels = Array(Channel(T | Exception)).new(tasks.size)

      tasks.each do |task|
        channel = Channel(T | Exception).new
        channels << channel

        spawn do
          semaphore.send(nil)
          begin
            result = task.call
            channel.send(result)
          rescue ex
            channel.send(ex)
          ensure
            semaphore.receive
          end
        end
      end

      results = Array(T).new(tasks.size)
      channels.each_with_index do |channel, _i|
        result = channel.receive
        if result.is_a?(Exception)
          raise result
        else
          results << result
        end
      end

      results
    end
  end
end
