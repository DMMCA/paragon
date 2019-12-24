package resolve

import (
	"context"
	"time"

	"github.com/kcarretto/paragon/ent"
	"github.com/kcarretto/paragon/ent/predicate"
	"github.com/kcarretto/paragon/ent/tag"
	"github.com/kcarretto/paragon/ent/target"
	"github.com/kcarretto/paragon/graphql/generated"
	"github.com/kcarretto/paragon/graphql/models"
)

// THIS CODE IS A STARTING POINT ONLY. IT WILL NOT BE UPDATED WITH SCHEMA CHANGES.

// Resolver is the root struct for handling all resolves
type Resolver struct {
	EntClient *ent.Client
}

// Job is the Resolver for the Job Ent
func (r *Resolver) Job() generated.JobResolver {
	return &jobResolver{r}
}

// Mutation is the Resolver for all Mutations
func (r *Resolver) Mutation() generated.MutationResolver {
	return &mutationResolver{r}
}

// Query is the Resolver for all Queries
func (r *Resolver) Query() generated.QueryResolver {
	return &queryResolver{r}
}

// Tag is the Resolver for the Tag Ent
func (r *Resolver) Tag() generated.TagResolver {
	return &tagResolver{r}
}

// Target is the Resolver for the Target Ent
func (r *Resolver) Target() generated.TargetResolver {
	return &targetResolver{r}
}

// Task is the Resolver for the Task Ent
func (r *Resolver) Task() generated.TaskResolver {
	return &taskResolver{r}
}

type jobResolver struct{ *Resolver }

func (r *jobResolver) Tasks(ctx context.Context, obj *ent.Job) ([]*ent.Task, error) {
	return obj.QueryTasks().All(ctx)
}
func (r *jobResolver) Tags(ctx context.Context, obj *ent.Job) ([]*ent.Tag, error) {
	return obj.QueryTags().All(ctx)
}
func (r *jobResolver) Next(ctx context.Context, obj *ent.Job) (*ent.Job, error) {
	return obj.QueryNext().Only(ctx)
}
func (r *jobResolver) Prev(ctx context.Context, obj *ent.Job) (*ent.Job, error) {
	return obj.QueryNext().Only(ctx)
}

type mutationResolver struct{ *Resolver }

func (r *mutationResolver) FailCredential(ctx context.Context, input *models.FailCredentialRequest) (bool, error) {
	_, err := r.EntClient.Credential.GetX(ctx, input.ID).
		Update().
		AddFails(1).
		Save(ctx)
	return err != nil, err
}
func (r *mutationResolver) CreateJob(ctx context.Context, input *models.CreateJobRequest) (int, error) {
	jobCreator := r.EntClient.Job.Create().
		SetName(input.Name).
		SetContent(input.Content).
		AddTagIDs(input.Tags...)
	if input.Prev != nil {
		jobCreator.SetPrevID(*input.Prev)
	}
	var tagPredicates []predicate.Target
	for _, tagID := range input.Tags {
		tagPredicates = append(tagPredicates, target.HasTagsWith(tag.ID(tagID)))
	}
	targets, err := r.EntClient.Target.Query().
		Where(target.And(tagPredicates...)).
		All(ctx)
	if err != nil {
		return -1, err
	}
	// just so all tasks and job are the same time.
	currentTime := time.Now()

	job, err := jobCreator.
		SetCreationTime(currentTime).
		Save(ctx)
	if err != nil {
		return -1, err
	}

	for _, target := range targets {
		task := r.EntClient.Task.Create().
			SetQueueTime(currentTime).
			SetContent(input.Content).
			AddTagIDs(input.Tags...).
			SetJobID(job.ID).
			SaveX(ctx)
		target.Update().
			AddTaskIDs(task.ID).
			SaveX(ctx)
	}

	return job.ID, nil
}
func (r *mutationResolver) CreateTag(ctx context.Context, input *models.CreateTagRequest) (int, error) {
	tag, err := r.EntClient.Tag.Create().
		SetName(input.Name).
		Save(ctx)
	if err != nil {
		return -1, err
	}
	return tag.ID, nil
}
func (r *mutationResolver) ApplyTagToTask(ctx context.Context, input *models.ApplyTagRequest) (bool, error) {
	_, err := r.EntClient.Task.UpdateOneID(input.EntID).
		AddTagIDs(input.TagID).
		Save(ctx)
	return err != nil, err
}
func (r *mutationResolver) ApplyTagToTarget(ctx context.Context, input *models.ApplyTagRequest) (bool, error) {
	_, err := r.EntClient.Target.UpdateOneID(input.EntID).
		AddTagIDs(input.TagID).
		Save(ctx)
	return err != nil, err
}
func (r *mutationResolver) ApplyTagToJob(ctx context.Context, input *models.ApplyTagRequest) (bool, error) {
	_, err := r.EntClient.Job.UpdateOneID(input.EntID).
		AddTagIDs(input.TagID).
		Save(ctx)
	return err != nil, err
}
func (r *mutationResolver) RemoveTagFromTask(ctx context.Context, input *models.RemoveTagRequest) (bool, error) {
	_, err := r.EntClient.Task.UpdateOneID(input.EntID).
		RemoveTagIDs(input.TagID).
		Save(ctx)
	return err != nil, err
}
func (r *mutationResolver) RemoveTagFromTarget(ctx context.Context, input *models.RemoveTagRequest) (bool, error) {
	_, err := r.EntClient.Target.UpdateOneID(input.EntID).
		RemoveTagIDs(input.TagID).
		Save(ctx)
	return err != nil, err
}
func (r *mutationResolver) RemoveTagFromJob(ctx context.Context, input *models.RemoveTagRequest) (bool, error) {
	_, err := r.EntClient.Job.UpdateOneID(input.EntID).
		RemoveTagIDs(input.TagID).
		Save(ctx)
	return err != nil, err
}
func (r *mutationResolver) CreateTarget(ctx context.Context, input *models.CreateTargetRequest) (int, error) {
	target, err := r.EntClient.Target.Create().
		SetName(input.Name).
		SetPrimaryIP(input.PrimaryIP).
		AddTagIDs(input.Tags...).
		Save(ctx)
	if err != nil {
		return -1, err
	}
	return target.ID, nil
}
func (r *mutationResolver) SetTargetFields(ctx context.Context, input *models.SetTargetFieldsRequest) (bool, error) {
	targetUpdater := r.EntClient.Target.UpdateOneID(input.ID)
	if input.Name != nil {
		targetUpdater.SetName(*input.Name)
	}
	if input.PrimaryIP != nil {
		targetUpdater.SetPrimaryIP(*input.PrimaryIP)
	}
	_, err := targetUpdater.
		SetNillableHostname(input.Hostname).
		SetNillableMachineUUID(input.MachineUUID).
		SetNillablePrimaryMAC(input.PrimaryMac).
		SetNillablePublicIP(input.PublicIP).
		Save(ctx)
	return err != nil, err
}
func (r *mutationResolver) DeleteTarget(ctx context.Context, input *models.DeleteTargetRequest) (bool, error) {
	err := r.EntClient.Target.DeleteOneID(input.ID).Exec(ctx)
	return err != nil, err
}
func (r *mutationResolver) AddCredentialForTarget(ctx context.Context, input *models.AddCredentialForTargetRequest) (bool, error) {
	credential, err := r.EntClient.Credential.Create().
		SetPrincipal(input.Principal).
		SetSecret(input.Secret).
		Save(ctx)
	if err != nil {
		return false, err
	}
	_, err = r.EntClient.Target.UpdateOneID(input.ID).
		AddCredentialIDs(credential.ID).
		Save(ctx)
	return err != nil, err
}

type queryResolver struct{ *Resolver }

func (r *queryResolver) Credential(ctx context.Context, id int) (*ent.Credential, error) {
	return r.EntClient.Credential.Get(ctx, id)
}
func (r *queryResolver) Credentials(ctx context.Context) ([]*ent.Credential, error) {
	return r.EntClient.Credential.Query().All(ctx)
}
func (r *queryResolver) Job(ctx context.Context, id int) (*ent.Job, error) {
	return r.EntClient.Job.Get(ctx, id)
}
func (r *queryResolver) Jobs(ctx context.Context) ([]*ent.Job, error) {
	return r.EntClient.Job.Query().All(ctx)
}
func (r *queryResolver) Tag(ctx context.Context, id int) (*ent.Tag, error) {
	return r.EntClient.Tag.Get(ctx, id)
}
func (r *queryResolver) Tags(ctx context.Context) ([]*ent.Tag, error) {
	return r.EntClient.Tag.Query().All(ctx)
}
func (r *queryResolver) Target(ctx context.Context, id int) (*ent.Target, error) {
	return r.EntClient.Target.Get(ctx, id)
}
func (r *queryResolver) Targets(ctx context.Context) ([]*ent.Target, error) {
	return r.EntClient.Target.Query().All(ctx)
}
func (r *queryResolver) Task(ctx context.Context, id int) (*ent.Task, error) {
	return r.EntClient.Task.Get(ctx, id)
}
func (r *queryResolver) Tasks(ctx context.Context) ([]*ent.Task, error) {
	return r.EntClient.Task.Query().All(ctx)
}

type tagResolver struct{ *Resolver }

func (r *tagResolver) Tasks(ctx context.Context, obj *ent.Tag) ([]*ent.Task, error) {
	return obj.QueryTasks().All(ctx)
}
func (r *tagResolver) Targets(ctx context.Context, obj *ent.Tag) ([]*ent.Target, error) {
	return obj.QueryTargets().All(ctx)
}
func (r *tagResolver) Jobs(ctx context.Context, obj *ent.Tag) ([]*ent.Job, error) {
	return obj.QueryJobs().All(ctx)
}

type targetResolver struct{ *Resolver }

func (r *targetResolver) Tasks(ctx context.Context, obj *ent.Target) ([]*ent.Task, error) {
	return obj.QueryTasks().All(ctx)
}
func (r *targetResolver) Tags(ctx context.Context, obj *ent.Target) ([]*ent.Tag, error) {
	return obj.QueryTags().All(ctx)
}
func (r *targetResolver) Credentials(ctx context.Context, obj *ent.Target) ([]*ent.Credential, error) {
	return obj.QueryCredentials().All(ctx)
}

type taskResolver struct{ *Resolver }

func (r *taskResolver) Job(ctx context.Context, obj *ent.Task) (*ent.Job, error) {
	return obj.QueryJob().Only(ctx)
}
