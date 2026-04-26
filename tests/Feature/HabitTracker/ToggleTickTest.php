<?php

namespace Tests\Feature\HabitTracker;

use App\Actions\HabitTracker\ToggleTick;
use App\Models\Habit;
use App\Models\Person;
use App\Models\Reward;
use App\Models\Sprint;
use App\Models\SprintParticipant;
use App\Models\Tick;
use Illuminate\Foundation\Testing\LazilyRefreshDatabase;
use Illuminate\Support\Carbon;
use Illuminate\Validation\ValidationException;
use Tests\TestCase;

class ToggleTickTest extends TestCase
{
    use LazilyRefreshDatabase;

    public function test_creates_a_tick_when_none_exists(): void
    {
        Carbon::setTestNow('2026-04-26 10:00:00');

        [$habit] = $this->makeActiveSprintWithHabit();

        $result = app(ToggleTick::class)->execute($habit, '2026-04-25');

        $this->assertNotNull($result['tick']);
        $this->assertSame(1, Tick::count());
        $this->assertSame(1, $result['balance_after']);
    }

    public function test_deletes_a_tick_when_one_exists(): void
    {
        Carbon::setTestNow('2026-04-26 10:00:00');

        [$habit] = $this->makeActiveSprintWithHabit();
        Tick::create(['habit_id' => $habit->id, 'tick_date' => '2026-04-25']);

        $result = app(ToggleTick::class)->execute($habit, '2026-04-25');

        $this->assertNull($result['tick']);
        $this->assertSame(0, Tick::count());
        $this->assertSame(0, $result['balance_after']);
    }

    public function test_double_toggle_is_idempotent(): void
    {
        Carbon::setTestNow('2026-04-26 10:00:00');

        [$habit] = $this->makeActiveSprintWithHabit();

        app(ToggleTick::class)->execute($habit, '2026-04-25');
        app(ToggleTick::class)->execute($habit, '2026-04-25');

        $this->assertSame(0, Tick::count());
    }

    public function test_rejects_future_day_tick(): void
    {
        Carbon::setTestNow('2026-04-26 10:00:00');

        [$habit] = $this->makeActiveSprintWithHabit();

        $this->expectException(ValidationException::class);
        app(ToggleTick::class)->execute($habit, '2026-04-30');
    }

    public function test_rejects_tick_outside_sprint_range(): void
    {
        Carbon::setTestNow('2026-04-26 10:00:00');

        [$habit] = $this->makeActiveSprintWithHabit();

        $this->expectException(ValidationException::class);
        app(ToggleTick::class)->execute($habit, '2026-03-01');
    }

    public function test_rejects_tick_on_archived_sprint(): void
    {
        Carbon::setTestNow('2026-04-26 10:00:00');

        [$habit, $sprint] = $this->makeActiveSprintWithHabit();
        $sprint->update(['status' => Sprint::STATUS_ARCHIVED, 'ended_at' => now()]);

        $this->expectException(ValidationException::class);
        app(ToggleTick::class)->execute($habit, '2026-04-25');
    }

    public function test_marks_reward_achieved_when_balance_reaches_cost(): void
    {
        Carbon::setTestNow('2026-04-26 10:00:00');

        [$habit, , $participant] = $this->makeActiveSprintWithHabit();

        $reward = Reward::create([
            'person_id' => $participant->person_id,
            'name' => 'Lego Set',
            'cost' => 2,
        ]);
        $participant->update(['active_reward_id' => $reward->id]);

        // First tick: balance = 1, not yet achieved.
        $r1 = app(ToggleTick::class)->execute($habit, '2026-04-24');
        $this->assertNull($r1['just_achieved_reward']);

        // Second tick: balance = 2, achievement triggered.
        $r2 = app(ToggleTick::class)->execute($habit, '2026-04-25');
        $this->assertNotNull($r2['just_achieved_reward']);
        $this->assertSame($reward->id, $r2['just_achieved_reward']->id);
        $this->assertNotNull($reward->refresh()->achieved_at);
    }

    /**
     * @return array{0: Habit, 1: Sprint, 2: SprintParticipant}
     */
    private function makeActiveSprintWithHabit(): array
    {
        $person = Person::factory()->create(['name' => 'Test', 'display_order' => 1]);
        $sprint = Sprint::factory()->create([
            'start_date' => '2026-04-19',
            'end_date' => '2026-05-16',
            'status' => Sprint::STATUS_ACTIVE,
            'started_at' => '2026-04-19 00:00:00',
        ]);
        $participant = SprintParticipant::factory()
            ->for($sprint)
            ->for($person)
            ->create();
        $habit = Habit::factory()->for($participant, 'sprintParticipant')->create();

        return [$habit, $sprint, $participant];
    }
}
