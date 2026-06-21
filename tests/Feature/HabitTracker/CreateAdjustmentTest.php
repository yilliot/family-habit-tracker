<?php

namespace Tests\Feature\HabitTracker;

use App\Actions\HabitTracker\CreateAdjustment;
use App\Models\Habit;
use App\Models\Person;
use App\Models\Reward;
use App\Models\Sprint;
use App\Models\SprintParticipant;
use App\Models\Tick;
use App\Support\HabitTracker\PointsCalculator;
use Illuminate\Foundation\Testing\LazilyRefreshDatabase;
use Illuminate\Support\Carbon;
use Illuminate\Validation\ValidationException;
use Tests\TestCase;

class CreateAdjustmentTest extends TestCase
{
    use LazilyRefreshDatabase;

    public function test_records_a_negative_adjustment_and_lowers_balance(): void
    {
        [$habit, , $participant] = $this->makeActiveSprintWithHabit();
        Tick::create(['habit_id' => $habit->id, 'tick_date' => '2026-04-20']);
        Tick::create(['habit_id' => $habit->id, 'tick_date' => '2026-04-21']);

        $points = app(PointsCalculator::class);
        $this->assertSame(2, $points->sprintParticipantBalance($participant->refresh()));

        $adjustment = app(CreateAdjustment::class)->execute($participant, -1, 'missed bedtime');

        $this->assertSame(-1, $adjustment->amount);
        $this->assertSame('missed bedtime', $adjustment->reason);
        $this->assertSame(1, $points->sprintParticipantBalance($participant->refresh()));
    }

    public function test_allows_balance_to_go_negative(): void
    {
        [, , $participant] = $this->makeActiveSprintWithHabit();

        app(CreateAdjustment::class)->execute($participant, -5);

        $this->assertSame(-5, app(PointsCalculator::class)->sprintParticipantBalance($participant->refresh()));
    }

    public function test_positive_amount_acts_as_a_manual_award(): void
    {
        [, , $participant] = $this->makeActiveSprintWithHabit();

        app(CreateAdjustment::class)->execute($participant, 3);

        $this->assertSame(3, app(PointsCalculator::class)->sprintParticipantBalance($participant->refresh()));
    }

    public function test_blank_reason_is_stored_as_null(): void
    {
        [, , $participant] = $this->makeActiveSprintWithHabit();

        $adjustment = app(CreateAdjustment::class)->execute($participant, -2, '   ');

        $this->assertNull($adjustment->reason);
    }

    public function test_adjustment_reduces_lifetime_balance_for_carry_forward(): void
    {
        [$habit, , $participant] = $this->makeActiveSprintWithHabit();
        Tick::create(['habit_id' => $habit->id, 'tick_date' => '2026-04-20']);

        app(CreateAdjustment::class)->execute($participant, -3);

        // 1 tick earned − 3 deducted = −2 lifetime.
        $this->assertSame(-2, app(PointsCalculator::class)->lifetimeBalanceForPerson($participant->person));
    }

    public function test_rejects_zero_amount(): void
    {
        [, , $participant] = $this->makeActiveSprintWithHabit();

        $this->expectException(ValidationException::class);
        app(CreateAdjustment::class)->execute($participant, 0);
    }

    public function test_rejects_adjustment_on_archived_sprint(): void
    {
        [, $sprint, $participant] = $this->makeActiveSprintWithHabit();
        $sprint->update(['status' => Sprint::STATUS_ARCHIVED, 'ended_at' => now()]);

        $this->expectException(ValidationException::class);
        app(CreateAdjustment::class)->execute($participant, -1);
    }

    public function test_adjustment_does_not_un_achieve_a_reward(): void
    {
        [$habit, , $participant] = $this->makeActiveSprintWithHabit();
        $reward = Reward::factory()
            ->for($participant->person)
            ->create(['cost' => 1, 'achieved_at' => now()]);
        $participant->update(['active_reward_id' => $reward->id]);

        app(CreateAdjustment::class)->execute($participant, -5);

        $this->assertNotNull($reward->refresh()->achieved_at);
    }

    /**
     * @return array{0: Habit, 1: Sprint, 2: SprintParticipant}
     */
    private function makeActiveSprintWithHabit(): array
    {
        Carbon::setTestNow('2026-04-26 10:00:00');

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
