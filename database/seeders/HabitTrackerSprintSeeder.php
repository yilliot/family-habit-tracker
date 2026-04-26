<?php

namespace Database\Seeders;

use App\Actions\HabitTracker\StartSprint;
use App\Models\Person;
use App\Models\Sprint;
use Illuminate\Database\Seeder;

class HabitTrackerSprintSeeder extends Seeder
{
    /**
     * Seed the Apr 19 – May 16, 2026 sprint with the four family members
     * and their habits from the printed Family Habit Tracker reference.
     *
     * Idempotent: skips if a sprint with this start date already exists.
     */
    public function run(StartSprint $startSprint): void
    {
        $startDate = '2026-04-19';
        $endDate = '2026-05-16';

        if (Sprint::query()->where('start_date', $startDate)->exists()) {
            return;
        }

        if (Sprint::query()->active()->exists()) {
            return;
        }

        $roster = [
            '爸爸' => [
                '整齐',
                '带敬拜',
                '11点半前睡觉',
            ],
            '妈妈' => [
                '聆听',
                '睡前祷告',
                '15分钟运动',
            ],
            '可遇' => [
                '整理书桌',
                '30分钟阅读时间',
                '半小时gymnastic',
            ],
            '奇乐' => [
                '整理床',
                '收拾game room',
                '华文练习',
            ],
        ];

        $participants = [];

        foreach ($roster as $personName => $habits) {
            $person = Person::query()->where('name', $personName)->first();

            if ($person === null) {
                continue;
            }

            $participants[] = [
                'person_id' => $person->id,
                'habits' => array_map(
                    fn (string $name, int $index) => ['name' => $name, 'display_order' => $index + 1],
                    $habits,
                    array_keys($habits),
                ),
                'reward' => null,
            ];
        }

        if ($participants === []) {
            return;
        }

        $startSprint->execute($startDate, $endDate, $participants);
    }
}
