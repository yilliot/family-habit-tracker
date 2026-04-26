<?php

namespace Tests\Feature\HabitTracker\Http;

use App\Models\Habit;
use App\Models\Person;
use App\Models\Reward;
use App\Models\Sprint;
use App\Models\SprintParticipant;
use App\Models\Tick;
use Illuminate\Foundation\Testing\LazilyRefreshDatabase;
use Illuminate\Support\Carbon;
use Tests\TestCase;

class TickControllerTest extends TestCase
{
    use LazilyRefreshDatabase;

    private function buildActiveSprintWithHabit(): Habit
    {
        Carbon::setTestNow('2026-04-26 10:00:00');

        $person = Person::factory()->create(['name' => 'Bob', 'display_order' => 1]);
        $sprint = Sprint::factory()->create([
            'start_date' => '2026-04-19',
            'end_date' => '2026-05-16',
            'started_at' => '2026-04-19 00:00:00',
        ]);
        $participant = SprintParticipant::factory()
            ->for($sprint)
            ->for($person)
            ->create();

        return Habit::factory()->for($participant, 'sprintParticipant')->create(['name' => 'Read']);
    }

    public function test_toggle_creates_tick_and_redirects_with_success_flash(): void
    {
        $habit = $this->buildActiveSprintWithHabit();

        $this->from(route('tracker.show'))
            ->post(route('ticks.toggle'), [
                'habit_id' => $habit->id,
                'tick_date' => '2026-04-20',
            ])
            ->assertRedirect(route('tracker.show'))
            ->assertSessionHas('success', 'Tick updated');

        $this->assertSame(
            1,
            Tick::query()->where('habit_id', $habit->id)->whereDate('tick_date', '2026-04-20')->count(),
        );
    }

    public function test_toggle_deletes_existing_tick(): void
    {
        $habit = $this->buildActiveSprintWithHabit();
        Tick::create(['habit_id' => $habit->id, 'tick_date' => '2026-04-20']);

        $this->from(route('tracker.show'))
            ->post(route('ticks.toggle'), [
                'habit_id' => $habit->id,
                'tick_date' => '2026-04-20',
            ])
            ->assertRedirect(route('tracker.show'));

        $this->assertSame(
            0,
            Tick::query()->where('habit_id', $habit->id)->whereDate('tick_date', '2026-04-20')->count(),
        );
    }

    public function test_toggle_rejects_future_date(): void
    {
        $habit = $this->buildActiveSprintWithHabit();

        $this->from(route('tracker.show'))
            ->post(route('ticks.toggle'), [
                'habit_id' => $habit->id,
                'tick_date' => '2026-05-01',
            ])
            ->assertSessionHasErrors('tick_date');
    }

    public function test_toggle_rejects_date_outside_sprint_range(): void
    {
        $habit = $this->buildActiveSprintWithHabit();

        $this->from(route('tracker.show'))
            ->post(route('ticks.toggle'), [
                'habit_id' => $habit->id,
                'tick_date' => '2026-04-10',
            ])
            ->assertSessionHasErrors('tick_date');
    }

    public function test_toggle_rejects_archived_sprint(): void
    {
        $habit = $this->buildActiveSprintWithHabit();
        $habit->sprintParticipant->sprint->update([
            'status' => Sprint::STATUS_ARCHIVED,
            'ended_at' => now(),
        ]);

        $this->from(route('tracker.show'))
            ->post(route('ticks.toggle'), [
                'habit_id' => $habit->id,
                'tick_date' => '2026-04-20',
            ])
            ->assertSessionHasErrors('sprint');
    }

    public function test_toggle_rejects_missing_habit(): void
    {
        $this->from(route('tracker.show'))
            ->post(route('ticks.toggle'), [
                'habit_id' => 999999,
                'tick_date' => '2026-04-20',
            ])
            ->assertSessionHasErrors('habit_id');
    }

    public function test_toggle_rejects_soft_deleted_habit(): void
    {
        $habit = $this->buildActiveSprintWithHabit();
        $habit->delete();

        $this->from(route('tracker.show'))
            ->post(route('ticks.toggle'), [
                'habit_id' => $habit->id,
                'tick_date' => '2026-04-20',
            ])
            ->assertSessionHasErrors('habit_id');
    }

    public function test_toggle_flashes_achievement_when_threshold_crossed(): void
    {
        Carbon::setTestNow('2026-04-26 10:00:00');

        $person = Person::factory()->create(['name' => 'Alice', 'display_order' => 1]);
        $sprint = Sprint::factory()->create([
            'start_date' => '2026-04-19',
            'end_date' => '2026-05-16',
            'started_at' => '2026-04-19 00:00:00',
        ]);
        $reward = Reward::factory()->for($person)->create(['name' => 'Lego', 'cost' => 1]);
        $participant = SprintParticipant::factory()
            ->for($sprint)
            ->for($person)
            ->create(['active_reward_id' => $reward->id]);
        $habit = Habit::factory()->for($participant, 'sprintParticipant')->create();

        $this->from(route('tracker.show'))
            ->post(route('ticks.toggle'), [
                'habit_id' => $habit->id,
                'tick_date' => '2026-04-20',
            ])
            ->assertRedirect(route('tracker.show'))
            ->assertSessionHas('achievement', fn ($achievement) => $achievement['person_id'] === $person->id
                && $achievement['reward_id'] === $reward->id
                && $achievement['person_name'] === 'Alice'
                && $achievement['reward_name'] === 'Lego'
            );
    }
}
