<?php

namespace Database\Factories;

use App\Models\Sprint;
use Illuminate\Database\Eloquent\Factories\Factory;

/**
 * @extends Factory<Sprint>
 */
class SprintFactory extends Factory
{
    protected $model = Sprint::class;

    /**
     * @return array<string, mixed>
     */
    public function definition(): array
    {
        $start = now()->startOfDay();
        $end = (clone $start)->addDays(27);

        return [
            'start_date' => $start->toDateString(),
            'end_date' => $end->toDateString(),
            'status' => Sprint::STATUS_ACTIVE,
            'started_at' => now(),
            'ended_at' => null,
        ];
    }

    public function archived(): static
    {
        return $this->state(fn (array $attributes): array => [
            'status' => Sprint::STATUS_ARCHIVED,
            'ended_at' => now(),
        ]);
    }
}
