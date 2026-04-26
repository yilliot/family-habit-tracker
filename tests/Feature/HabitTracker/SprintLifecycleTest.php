<?php

namespace Tests\Feature\HabitTracker;

use App\Actions\HabitTracker\AddParticipant;
use App\Actions\HabitTracker\EndSprint;
use App\Actions\HabitTracker\StartSprint;
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

class SprintLifecycleTest extends TestCase
{
    use LazilyRefreshDatabase;

    public function test_start_sprint_creates_sprint_with_participants_and_habits(): void
    {
        Carbon::setTestNow('2026-04-26 10:00:00');

        $father = Person::factory()->create(['name' => '爸爸', 'display_order' => 1]);
        $mother = Person::factory()->create(['name' => '妈妈', 'display_order' => 2]);

        $sprint = app(StartSprint::class)->execute(
            '2026-04-19',
            '2026-05-16',
            [
                [
                    'person_id' => $father->id,
                    'habits' => [['name' => 'Read 30min']],
                    'reward' => ['name' => 'Coffee Mug', 'cost' => 50],
                ],
                [
                    'person_id' => $mother->id,
                    'habits' => [['name' => 'Walk']],
                    'reward' => null,
                ],
            ]
        );

        $this->assertSame(Sprint::STATUS_ACTIVE, $sprint->status);
        $this->assertCount(2, $sprint->participants);
        $this->assertSame(1, $sprint->participants->first()->habits->count());
        $this->assertNotNull($sprint->participants->first()->active_reward_id);
    }

    public function test_two_active_sprints_blocked(): void
    {
        Carbon::setTestNow('2026-04-26 10:00:00');

        $person = Person::factory()->create();
        Sprint::factory()->create();

        $this->expectException(ValidationException::class);
        app(StartSprint::class)->execute('2026-04-19', '2026-05-16', [
            ['person_id' => $person->id, 'habits' => [], 'reward' => null],
        ]);
    }

    public function test_start_sprint_rejects_end_before_start(): void
    {
        Carbon::setTestNow('2026-04-26 10:00:00');

        $person = Person::factory()->create();

        $this->expectException(ValidationException::class);
        app(StartSprint::class)->execute('2026-05-01', '2026-04-19', [
            ['person_id' => $person->id, 'habits' => [], 'reward' => null],
        ]);
    }

    public function test_end_sprint_archives_it(): void
    {
        Carbon::setTestNow('2026-04-26 10:00:00');

        $sprint = Sprint::factory()->create();

        $archived = app(EndSprint::class)->execute($sprint);

        $this->assertSame(Sprint::STATUS_ARCHIVED, $archived->status);
        $this->assertNotNull($archived->ended_at);
    }

    public function test_end_sprint_when_no_reward_set_is_allowed(): void
    {
        Carbon::setTestNow('2026-04-26 10:00:00');

        $person = Person::factory()->create();
        $sprint = Sprint::factory()->create();
        SprintParticipant::factory()
            ->for($sprint)
            ->for($person)
            ->create(['active_reward_id' => null]);

        $archived = app(EndSprint::class)->execute($sprint);

        $this->assertSame(Sprint::STATUS_ARCHIVED, $archived->status);
    }

    public function test_carry_forward_seeds_next_sprint_with_unspent_balance(): void
    {
        Carbon::setTestNow('2026-04-26 10:00:00');

        $person = Person::factory()->create(['name' => 'Bob', 'display_order' => 1]);

        // First sprint: earn 5 ticks, no achieved reward.
        $sprint1 = Sprint::factory()->create([
            'start_date' => '2026-03-01',
            'end_date' => '2026-03-28',
            'started_at' => '2026-03-01 00:00:00',
        ]);
        $participant1 = SprintParticipant::factory()
            ->for($sprint1)
            ->for($person)
            ->create();
        $habit = Habit::factory()->for($participant1, 'sprintParticipant')->create();
        foreach (range(0, 4) as $i) {
            Tick::create([
                'habit_id' => $habit->id,
                'tick_date' => Carbon::parse('2026-03-01')->addDays($i)->toDateString(),
            ]);
        }
        app(EndSprint::class)->execute($sprint1);

        // Start a second sprint.
        $sprint2 = app(StartSprint::class)->execute('2026-04-01', '2026-04-28', [
            ['person_id' => $person->id, 'habits' => [['name' => 'Read']], 'reward' => null],
        ]);

        $newParticipant = $sprint2->participants->first();
        $this->assertSame(5, $newParticipant->carry_forward_balance);
    }

    public function test_carry_forward_zero_when_balance_was_spent(): void
    {
        Carbon::setTestNow('2026-04-26 10:00:00');

        $person = Person::factory()->create();

        // Sprint 1: earn 50 ticks, achieve a 50-cost reward.
        $sprint1 = Sprint::factory()->create([
            'start_date' => '2026-03-01',
            'end_date' => '2026-03-28',
            'started_at' => '2026-03-01 00:00:00',
        ]);
        $reward = Reward::factory()->for($person)->achieved()->create(['cost' => 5]);
        $participant1 = SprintParticipant::factory()
            ->for($sprint1)
            ->for($person)
            ->create(['active_reward_id' => $reward->id]);
        $habit = Habit::factory()->for($participant1, 'sprintParticipant')->create();
        foreach (range(0, 4) as $i) {
            Tick::create([
                'habit_id' => $habit->id,
                'tick_date' => Carbon::parse('2026-03-01')->addDays($i)->toDateString(),
            ]);
        }
        app(EndSprint::class)->execute($sprint1);

        $sprint2 = app(StartSprint::class)->execute('2026-04-01', '2026-04-28', [
            ['person_id' => $person->id, 'habits' => [], 'reward' => null],
        ]);

        $this->assertSame(0, $sprint2->participants->first()->carry_forward_balance);
    }

    public function test_add_participant_mid_sprint(): void
    {
        Carbon::setTestNow('2026-04-26 10:00:00');

        $sprint = Sprint::factory()->create();
        $person = Person::factory()->create();

        $participant = app(AddParticipant::class)->execute(
            $sprint,
            $person->id,
            [['name' => 'Hydrate']],
            ['name' => 'Treat', 'cost' => 20]
        );

        $this->assertSame($sprint->id, $participant->sprint_id);
        $this->assertSame($person->id, $participant->person_id);
        $this->assertSame(1, $participant->habits->count());
        $this->assertNotNull($participant->active_reward_id);
    }

    public function test_add_participant_blocked_on_archived_sprint(): void
    {
        $sprint = Sprint::factory()->archived()->create();
        $person = Person::factory()->create();

        $this->expectException(ValidationException::class);
        app(AddParticipant::class)->execute($sprint, $person->id);
    }

    public function test_add_participant_rejects_duplicate(): void
    {
        $sprint = Sprint::factory()->create();
        $person = Person::factory()->create();
        SprintParticipant::factory()->for($sprint)->for($person)->create();

        $this->expectException(ValidationException::class);
        app(AddParticipant::class)->execute($sprint, $person->id);
    }
}
