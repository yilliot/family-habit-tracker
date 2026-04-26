<?php

namespace Database\Factories;

use App\Models\Habit;
use App\Models\Tick;
use Illuminate\Database\Eloquent\Factories\Factory;

/**
 * @extends Factory<Tick>
 */
class TickFactory extends Factory
{
    protected $model = Tick::class;

    /**
     * @return array<string, mixed>
     */
    public function definition(): array
    {
        return [
            'habit_id' => Habit::factory(),
            'tick_date' => now()->toDateString(),
            'created_at' => now(),
        ];
    }
}
