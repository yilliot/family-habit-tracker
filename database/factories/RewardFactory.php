<?php

namespace Database\Factories;

use App\Models\Person;
use App\Models\Reward;
use Illuminate\Database\Eloquent\Factories\Factory;

/**
 * @extends Factory<Reward>
 */
class RewardFactory extends Factory
{
    protected $model = Reward::class;

    /**
     * @return array<string, mixed>
     */
    public function definition(): array
    {
        return [
            'person_id' => Person::factory(),
            'name' => fake()->words(2, true),
            'cost' => fake()->numberBetween(20, 250),
            'achieved_at' => null,
        ];
    }

    public function achieved(): static
    {
        return $this->state(fn (array $attributes): array => [
            'achieved_at' => now(),
        ]);
    }
}
