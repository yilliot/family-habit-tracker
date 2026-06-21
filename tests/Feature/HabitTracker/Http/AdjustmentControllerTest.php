<?php

namespace Tests\Feature\HabitTracker\Http;

use App\Models\Person;
use App\Models\PointAdjustment;
use App\Models\Sprint;
use App\Models\SprintParticipant;
use Illuminate\Foundation\Testing\LazilyRefreshDatabase;
use Illuminate\Support\Carbon;
use Tests\TestCase;

class AdjustmentControllerTest extends TestCase
{
    use LazilyRefreshDatabase;

    private function makeParticipant(): SprintParticipant
    {
        Carbon::setTestNow('2026-04-26 10:00:00');

        $person = Person::factory()->create(['name' => 'Bob', 'display_order' => 1]);
        $sprint = Sprint::factory()->create([
            'start_date' => '2026-04-19',
            'end_date' => '2026-05-16',
            'started_at' => '2026-04-19 00:00:00',
        ]);

        return SprintParticipant::factory()->for($sprint)->for($person)->create();
    }

    public function test_deduct_records_a_negative_adjustment_and_redirects_with_success(): void
    {
        $participant = $this->makeParticipant();

        $this->from(route('tracker.show'))
            ->post(route('adjustments.store', $participant), [
                'points' => 3,
                'reason' => 'left toys out',
            ])
            ->assertRedirect(route('tracker.show'))
            ->assertSessionHas('success', 'Points deducted');

        $this->assertDatabaseHas('point_adjustments', [
            'sprint_participant_id' => $participant->id,
            'amount' => -3,
            'reason' => 'left toys out',
        ]);
    }

    public function test_reason_is_optional(): void
    {
        $participant = $this->makeParticipant();

        $this->from(route('tracker.show'))
            ->post(route('adjustments.store', $participant), ['points' => 2])
            ->assertRedirect(route('tracker.show'));

        $this->assertDatabaseHas('point_adjustments', [
            'sprint_participant_id' => $participant->id,
            'amount' => -2,
            'reason' => null,
        ]);
    }

    public function test_points_are_required(): void
    {
        $participant = $this->makeParticipant();

        $this->from(route('tracker.show'))
            ->post(route('adjustments.store', $participant), [])
            ->assertSessionHasErrors('points');

        $this->assertSame(0, PointAdjustment::count());
    }

    public function test_points_must_be_at_least_one(): void
    {
        $participant = $this->makeParticipant();

        $this->from(route('tracker.show'))
            ->post(route('adjustments.store', $participant), ['points' => 0])
            ->assertSessionHasErrors('points');
    }

    public function test_points_must_not_exceed_max(): void
    {
        $participant = $this->makeParticipant();

        $this->from(route('tracker.show'))
            ->post(route('adjustments.store', $participant), ['points' => 10000])
            ->assertSessionHasErrors('points');
    }

    public function test_deduct_on_archived_sprint_is_rejected(): void
    {
        $participant = $this->makeParticipant();
        $participant->sprint->update([
            'status' => Sprint::STATUS_ARCHIVED,
            'ended_at' => now(),
        ]);

        $this->from(route('tracker.show'))
            ->post(route('adjustments.store', $participant), ['points' => 1])
            ->assertSessionHasErrors('sprint');

        $this->assertSame(0, PointAdjustment::count());
    }
}
